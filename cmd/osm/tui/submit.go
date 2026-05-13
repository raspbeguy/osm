package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
)

type submitState int

const (
	submitFocusComment submitState = iota
	submitFocusTags
	submitAddingTag
	submitSending
	submitDone
)

type changesetSubmittedMsg struct {
	csID      osm.ChangesetID
	uploadErr error
	closeErr  error
}

type submitConfirmedMsg struct{}

type submitTagItem struct{ t osm.Tag }

func (i submitTagItem) Title() string       { return i.t.Key + " = " + i.t.Value }
func (i submitTagItem) Description() string { return "" }
func (i submitTagItem) FilterValue() string { return i.t.Key }

type submitChangesetModel struct {
	client       *api.Client
	staged       []*stagedElement
	commentInput textinput.Model
	tagInput     textinput.Model
	tagsList     list.Model
	customTags   osm.Tags
	state        submitState
	spinner      spinner.Model
	resultID     osm.ChangesetID
	uploadErr    error
	closeErr     error
	err          error
}

func newSubmit(c *api.Client) submitChangesetModel {
	ci := textinput.New()
	ci.Placeholder = "comment (required)"
	ci.CharLimit = 255
	ti := textinput.New()
	ti.Placeholder = "key=value"
	ti.CharLimit = 200
	l := list.New(nil, newCompactDelegate(), 60, 10)
	l.Title = "Additional tags"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	s := spinner.New()
	s.Spinner = spinner.Dot
	return submitChangesetModel{
		client:       c,
		commentInput: ci,
		tagInput:     ti,
		tagsList:     l,
		spinner:      s,
	}
}

func (m submitChangesetModel) Init() tea.Cmd { return nil }

func (m submitChangesetModel) show(staged []*stagedElement) (submitChangesetModel, tea.Cmd) {
	m.staged = staged
	m.commentInput.SetValue("")
	m.commentInput.Focus()
	m.customTags = nil
	m.refreshTags()
	m.state = submitFocusComment
	m.err = nil
	m.uploadErr = nil
	m.closeErr = nil
	m.resultID = 0
	return m, textinput.Blink
}

func (m *submitChangesetModel) refreshTags() {
	items := make([]list.Item, len(m.customTags))
	for i, t := range m.customTags {
		items[i] = submitTagItem{t: t}
	}
	m.tagsList.SetItems(items)
}

func (m submitChangesetModel) Update(msg tea.Msg) (submitChangesetModel, tea.Cmd) {
	switch msg := msg.(type) {
	case changesetSubmittedMsg:
		m.state = submitDone
		m.resultID = msg.csID
		m.uploadErr = msg.uploadErr
		m.closeErr = msg.closeErr
		if msg.uploadErr != nil {
			m.err = msg.uploadErr
		} else {
			m.err = msg.closeErr
		}
		return m, nil
	case spinner.TickMsg:
		if m.state != submitSending {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch m.state {
		case submitSending:
			return m, nil
		case submitDone:
			if msg.String() == "enter" && m.err == nil {
				return m, func() tea.Msg { return submitConfirmedMsg{} }
			}
			return m, nil
		case submitAddingTag:
			switch msg.String() {
			case "enter":
				m = m.commitTagInput()
				return m, nil
			case "esc":
				m.state = submitFocusTags
				m.err = nil
				return m, nil
			}
			var cmd tea.Cmd
			m.tagInput, cmd = m.tagInput.Update(msg)
			return m, cmd
		case submitFocusComment:
			switch msg.String() {
			case "tab":
				m.state = submitFocusTags
				m.commentInput.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.commentInput, cmd = m.commentInput.Update(msg)
			return m, cmd
		case submitFocusTags:
			switch msg.String() {
			case "tab":
				m.state = submitFocusComment
				m.commentInput.Focus()
				return m, textinput.Blink
			case "a":
				m.state = submitAddingTag
				m.tagInput.SetValue("")
				m.tagInput.Focus()
				m.err = nil
				return m, textinput.Blink
			case "d":
				idx := m.tagsList.Index()
				if idx >= 0 && idx < len(m.customTags) {
					m.customTags = append(m.customTags[:idx], m.customTags[idx+1:]...)
					m.refreshTags()
				}
				return m, nil
			case "s":
				cmt := strings.TrimSpace(m.commentInput.Value())
				if cmt == "" {
					m.err = errors.New("comment is required")
					return m, nil
				}
				return m.beginSubmit(cmt)
			}
			var cmd tea.Cmd
			m.tagsList, cmd = m.tagsList.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m submitChangesetModel) commitTagInput() submitChangesetModel {
	v := strings.TrimSpace(m.tagInput.Value())
	i := strings.IndexByte(v, '=')
	if i < 0 {
		m.err = errors.New("expected key=value")
		return m
	}
	key := strings.TrimSpace(v[:i])
	val := strings.TrimSpace(v[i+1:])
	if key == "" {
		m.err = errors.New("key required")
		return m
	}
	if key == "comment" || key == "created_by" {
		m.err = errors.New("comment and created_by are set automatically")
		return m
	}
	m.customTags = append(m.customTags, osm.Tag{Key: key, Value: val})
	m.refreshTags()
	m.state = submitFocusTags
	m.err = nil
	return m
}

func (m submitChangesetModel) beginSubmit(comment string) (submitChangesetModel, tea.Cmd) {
	tags := osm.Tags{
		{Key: "created_by", Value: "osm-tui " + osmTUIVersion},
		{Key: "comment", Value: comment},
	}
	tags = append(tags, m.customTags...)
	m.state = submitSending
	m.err = nil
	m.uploadErr = nil
	m.closeErr = nil
	client := m.client
	staged := cloneStaged(m.staged)
	return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
		csID, uploadErr, closeErr := submitChangeset(client, tags, staged)
		return changesetSubmittedMsg{csID: csID, uploadErr: uploadErr, closeErr: closeErr}
	})
}

// cloneStaged deep-copies the staged slice so a concurrent edit in compose
// can't mutate the payload while the submit goroutine is reading it.
func cloneStaged(in []*stagedElement) []*stagedElement {
	out := make([]*stagedElement, len(in))
	for i, e := range in {
		cp := *e
		cp.Tags = append(osm.Tags(nil), e.Tags...)
		cp.Nodes = append(osm.WayNodes(nil), e.Nodes...)
		cp.Members = append(osm.Members(nil), e.Members...)
		out[i] = &cp
	}
	return out
}

func submitChangeset(c *api.Client, tags osm.Tags, staged []*stagedElement) (osm.ChangesetID, error, error) {
	ctx := programCtx
	csID, err := c.OpenChangeset(ctx, tags)
	if err != nil {
		return 0, fmt.Errorf("open changeset: %w", err), nil
	}
	change := buildChange(staged)
	var uploadErr error
	if _, err := c.UploadChange(ctx, csID, change); err != nil {
		uploadErr = fmt.Errorf("upload change: %w", err)
	}
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	var closeErr error
	if err := c.CloseChangeset(closeCtx, csID); err != nil {
		closeErr = fmt.Errorf("close changeset: %w", err)
	}
	return csID, uploadErr, closeErr
}

func buildChange(staged []*stagedElement) *osm.Change {
	change := &osm.Change{
		Version:   "0.6",
		Generator: "osm-tui " + osmTUIVersion,
	}
	ensureCreate := func() *osm.OSM {
		if change.Create == nil {
			change.Create = &osm.OSM{}
		}
		return change.Create
	}
	ensureModify := func() *osm.OSM {
		if change.Modify == nil {
			change.Modify = &osm.OSM{}
		}
		return change.Modify
	}
	for _, e := range staged {
		target := ensureModify()
		if e.Action == stagedCreate {
			target = ensureCreate()
		}
		switch e.Kind {
		case "node":
			target.Nodes = append(target.Nodes, &osm.Node{
				ID: osm.NodeID(e.ID), Version: e.Version, Visible: true,
				Lat: e.Lat, Lon: e.Lon, Tags: e.Tags,
			})
		case "way":
			target.Ways = append(target.Ways, &osm.Way{
				ID: osm.WayID(e.ID), Version: e.Version, Visible: true,
				Nodes: e.Nodes, Tags: e.Tags,
			})
		case "relation":
			target.Relations = append(target.Relations, &osm.Relation{
				ID: osm.RelationID(e.ID), Version: e.Version, Visible: true,
				Tags: e.Tags, Members: e.Members,
			})
		}
	}
	return change
}

func (m submitChangesetModel) View() string {
	if m.state == submitSending {
		return m.spinner.View() + " submitting changeset..."
	}
	if m.state == submitDone {
		if m.uploadErr == nil && m.closeErr == nil {
			return headerStyle.Render(fmt.Sprintf("changeset %d created", m.resultID)) + "\n\n" +
				mutedStyle.Render("staged changes will be cleared on continue") + "\n" +
				footerStyle.Render("enter continue")
		}
		var head string
		switch {
		case m.resultID == 0:
			head = errorStyle.Render("submit failed: " + m.uploadErr.Error())
		case m.uploadErr != nil && m.closeErr != nil:
			head = errorStyle.Render("submit failed: "+m.uploadErr.Error()) + "\n" +
				errorStyle.Render("close also failed: "+m.closeErr.Error()) + "\n" +
				mutedStyle.Render(fmt.Sprintf("changeset %d may still be open on the server", m.resultID))
		case m.uploadErr != nil:
			head = errorStyle.Render("upload failed: "+m.uploadErr.Error()) + "\n" +
				mutedStyle.Render(fmt.Sprintf("changeset %d was closed without changes", m.resultID))
		default:
			head = errorStyle.Render("close failed: "+m.closeErr.Error()) + "\n" +
				mutedStyle.Render(fmt.Sprintf("changeset %d uploaded but may still be open on the server", m.resultID))
		}
		return head + "\n\n" +
			mutedStyle.Render("staged changes are preserved") + "\n" +
			footerStyle.Render("esc back")
	}
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Submit changeset") + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("%d staged element(s)", len(m.staged))) + "\n\n")
	commentLabel := "Comment (required)"
	if m.state == submitFocusComment {
		commentLabel = "▶ " + commentLabel
	}
	sb.WriteString(headerStyle.Render(commentLabel) + "\n")
	sb.WriteString(m.commentInput.View() + "\n\n")
	tagsLabel := "Additional tags"
	if m.state == submitFocusTags || m.state == submitAddingTag {
		tagsLabel = "▶ " + tagsLabel
	}
	sb.WriteString(headerStyle.Render(tagsLabel) + "\n")
	if m.state == submitAddingTag {
		sb.WriteString(m.tagInput.View() + "\n")
	} else if len(m.customTags) == 0 {
		sb.WriteString(mutedStyle.Render("  (none)") + "\n")
	} else {
		sb.WriteString(m.tagsList.View() + "\n")
	}
	sb.WriteString("\n" + mutedStyle.Render("created_by=osm-tui "+osmTUIVersion+" is added automatically") + "\n")

	var footer string
	switch m.state {
	case submitFocusComment:
		footer = "tab focus tags, esc cancel"
	case submitFocusTags:
		footer = "tab focus comment, a add tag, d delete tag, s submit, esc cancel"
	case submitAddingTag:
		footer = "enter save, esc cancel"
	}
	if m.err != nil {
		footer = errorStyle.Render(m.err.Error()) + " - " + footer
	}
	return sb.String() + "\n" + footerStyle.Render(footer)
}
