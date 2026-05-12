package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
)

// osmTUIVersion is stamped into the auto-added created_by tag on submission.
const osmTUIVersion = "0.1.0"

type stagedAction int

const (
	stagedModify stagedAction = iota
	stagedCreate
)

type stagedElement struct {
	Kind    string
	ID      int64
	Version int
	Action  stagedAction
	Tags    osm.Tags
	Lat     float64
	Lon     float64
	Nodes   osm.WayNodes
	Members osm.Members
}

type stagedAddedMsg struct {
	elem *stagedElement
}

type stagedItem struct {
	idx int
	e   *stagedElement
}

func (i stagedItem) Title() string {
	sign := "~"
	if i.e.Action == stagedCreate {
		sign = "+"
	}
	return fmt.Sprintf("%s %s %d", sign, kindGlyph(i.e.Kind), i.e.ID)
}

func (i stagedItem) Description() string { return "" }

func (i stagedItem) FilterValue() string { return i.Title() }

type composeChangesetModel struct {
	client   *api.Client
	list     list.Model
	viewport viewport.Model
	staged   []*stagedElement
	err      error
	focus    int // 0=list, 1=detail
	lastIdx  int
}

func newCompose(c *api.Client) composeChangesetModel {
	l := list.New(nil, newCompactDelegate(), 40, 20)
	l.Title = "New changeset"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return composeChangesetModel{client: c, list: l, viewport: viewport.New(40, 20), lastIdx: -1}
}

func (m composeChangesetModel) Init() tea.Cmd { return nil }

func (m composeChangesetModel) show() (composeChangesetModel, tea.Cmd) {
	m.err = nil
	m.refreshList()
	m = m.rewrap()
	return m, nil
}

func (m composeChangesetModel) rewrap() composeChangesetModel {
	sel := m.selectedStaged()
	if sel == nil {
		m.viewport.SetContent(mutedStyle.Render("(no selection)"))
		return m
	}
	m.viewport.SetContent(wrapText(renderStagedElement(sel), m.viewport.Width))
	return m
}

func (m *composeChangesetModel) refreshList() {
	items := make([]list.Item, len(m.staged))
	for i, e := range m.staged {
		items[i] = stagedItem{idx: i, e: e}
	}
	m.list.SetItems(items)
}

func (m composeChangesetModel) selectedStaged() *stagedElement {
	idx := m.list.Index()
	if idx < 0 || idx >= len(m.staged) {
		return nil
	}
	return m.staged[idx]
}

func (m composeChangesetModel) Update(msg tea.Msg) (composeChangesetModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "a":
			return m, func() tea.Msg {
				return navigateMsg{to: screenAddElement}
			}
		case "n":
			e := &stagedElement{Kind: "relation", ID: nextNewID(m.staged), Action: stagedCreate}
			m.staged = append(m.staged, e)
			m.refreshList()
			m.list.Select(len(m.staged) - 1)
			m = m.rewrap()
			return m, func() tea.Msg {
				return navigateMsg{to: screenEditElement}
			}
		case "enter":
			if m.selectedStaged() != nil {
				return m, func() tea.Msg {
					return navigateMsg{to: screenEditElement}
				}
			}
		case "d":
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.staged) {
				m.staged = append(m.staged[:idx], m.staged[idx+1:]...)
				m.refreshList()
				m = m.rewrap()
			}
			return m, nil
		case "c":
			m.staged = nil
			m.refreshList()
			m = m.rewrap()
			return m, nil
		case "tab":
			m.focus = 1 - m.focus
			return m, nil
		case "s":
			if len(m.staged) == 0 {
				m.err = errors.New("nothing to submit")
				return m, nil
			}
			return m, func() tea.Msg {
				return navigateMsg{to: screenSubmitChangeset}
			}
		}
	}
	prevIdx := m.list.Index()
	var cmd tea.Cmd
	if m.focus == 0 {
		m.list, cmd = m.list.Update(msg)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	if m.list.Index() != prevIdx {
		m = m.rewrap()
	}
	return m, cmd
}

func (m composeChangesetModel) View() string {
	footer := "esc back, a add by id, n new relation, s submit"
	if len(m.staged) > 0 {
		footer = "esc back, tab swap pane, a add, n new relation, enter edit, d drop, c clear, s submit"
	}
	if m.err != nil {
		footer = errorStyle.Render(m.err.Error()) + " - " + footer
	}
	if len(m.staged) == 0 {
		body := mutedStyle.Render("(no staged changes yet)") + "\n\n" +
			mutedStyle.Render("press 'a' to add an existing element, 'n' to create a new relation")
		return body + "\n" + footerStyle.Render(footer)
	}
	leftStyle, rightStyle := paneFocused, paneUnfocused
	if m.focus == 1 {
		leftStyle, rightStyle = paneUnfocused, paneFocused
	}
	left := leftStyle.Render(m.list.View())
	right := rightStyle.Render(m.viewport.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return body + "\n" + footerStyle.Render(footer)
}

func renderStagedElement(e *stagedElement) string {
	var sb strings.Builder
	action := "modify"
	if e.Action == stagedCreate {
		action = "create"
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("%s %s %d", action, kindGlyph(e.Kind), e.ID)) + "\n")
	if e.Version > 0 {
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("version %d", e.Version)) + "\n")
	}
	sb.WriteString("\n" + headerStyle.Render("Tags") + "\n")
	if len(e.Tags) == 0 {
		sb.WriteString(mutedStyle.Render("  (none)") + "\n")
	} else {
		for _, t := range e.Tags {
			fmt.Fprintf(&sb, "  %s = %s\n", t.Key, t.Value)
		}
	}
	if e.Kind == "relation" {
		sb.WriteString("\n" + headerStyle.Render("Members") + "\n")
		if len(e.Members) == 0 {
			sb.WriteString(mutedStyle.Render("  (none)") + "\n")
		} else {
			for _, mm := range e.Members {
				role := mm.Role
				if role == "" {
					role = "(no role)"
				}
				fmt.Fprintf(&sb, "  %s %d  [%s]\n", kindGlyph(string(mm.Type)), mm.Ref, role)
			}
		}
	}
	return sb.String()
}

func nextNewID(staged []*stagedElement) int64 {
	lowest := int64(0)
	for _, e := range staged {
		if e.ID < lowest {
			lowest = e.ID
		}
	}
	return lowest - 1
}

// fetchStagedFor builds a *stagedElement from the current server state of an
// element. Used by the add-by-id flow.
func fetchStagedFor(c *api.Client, kind string, id int64) (*stagedElement, error) {
	ctx := programCtx
	switch kind {
	case "node":
		n, err := c.GetNode(ctx, osm.NodeID(id))
		if err != nil {
			return nil, err
		}
		return &stagedElement{
			Kind: "node", ID: int64(n.ID), Version: n.Version,
			Action: stagedModify,
			Tags:   n.Tags, Lat: n.Lat, Lon: n.Lon,
		}, nil
	case "way":
		w, err := c.GetWay(ctx, osm.WayID(id))
		if err != nil {
			return nil, err
		}
		return &stagedElement{
			Kind: "way", ID: int64(w.ID), Version: w.Version,
			Action: stagedModify,
			Tags:   w.Tags, Nodes: w.Nodes,
		}, nil
	case "relation":
		r, err := c.GetRelation(ctx, osm.RelationID(id))
		if err != nil {
			return nil, err
		}
		return &stagedElement{
			Kind: "relation", ID: int64(r.ID), Version: r.Version,
			Action: stagedModify,
			Tags:   r.Tags, Members: r.Members,
		}, nil
	}
	return nil, fmt.Errorf("unknown kind %q", kind)
}
