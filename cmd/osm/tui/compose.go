package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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
	return fmt.Sprintf("%s %s %d", sign, i.e.Kind, i.e.ID)
}

func (i stagedItem) Description() string {
	if len(i.e.Tags) == 0 {
		return "(no tags)"
	}
	parts := make([]string, 0, len(i.e.Tags))
	for _, t := range i.e.Tags {
		parts = append(parts, t.Key+"="+t.Value)
	}
	return strings.Join(parts, ", ")
}

func (i stagedItem) FilterValue() string { return i.Title() }

type composeChangesetModel struct {
	client *api.Client
	list   list.Model
	staged []*stagedElement
	err    error
}

func newCompose(c *api.Client) composeChangesetModel {
	l := list.New(nil, list.NewDefaultDelegate(), 60, 20)
	l.Title = "New changeset"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return composeChangesetModel{client: c, list: l}
}

func (m composeChangesetModel) Init() tea.Cmd { return nil }

func (m composeChangesetModel) show() (composeChangesetModel, tea.Cmd) {
	m.err = nil
	m.refreshList()
	return m, nil
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
		switch msg.String() {
		case "a":
			return m, func() tea.Msg {
				return navigateMsg{to: screenAddElement, parent: screenComposeChangeset}
			}
		case "n":
			e := &stagedElement{Kind: "relation", ID: nextNewID(m.staged), Action: stagedCreate}
			m.staged = append(m.staged, e)
			m.refreshList()
			m.list.Select(len(m.staged) - 1)
			return m, func() tea.Msg {
				return navigateMsg{to: screenEditElement, parent: screenComposeChangeset}
			}
		case "enter":
			if m.selectedStaged() != nil {
				return m, func() tea.Msg {
					return navigateMsg{to: screenEditElement, parent: screenComposeChangeset}
				}
			}
		case "d":
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.staged) {
				m.staged = append(m.staged[:idx], m.staged[idx+1:]...)
				m.refreshList()
			}
			return m, nil
		case "c":
			m.staged = nil
			m.refreshList()
			return m, nil
		case "s":
			if len(m.staged) == 0 {
				m.err = errors.New("nothing to submit")
				return m, nil
			}
			return m, func() tea.Msg {
				return navigateMsg{to: screenSubmitChangeset, parent: screenComposeChangeset}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m composeChangesetModel) View() string {
	var body string
	if len(m.staged) == 0 {
		body = mutedStyle.Render("(no staged changes yet)")
	} else {
		body = m.list.View()
	}
	footer := "esc back, a add by id, n new relation, enter edit, d drop, c clear, s submit"
	if m.err != nil {
		footer = errorStyle.Render(m.err.Error()) + " — " + footer
	}
	return body + "\n" + footerStyle.Render(footer)
}

func nextNewID(staged []*stagedElement) int64 {
	min := int64(0)
	for _, e := range staged {
		if e.ID < min {
			min = e.ID
		}
	}
	return min - 1
}

// fetchStagedFor builds a *stagedElement from the current server state of an
// element. Used by the add-by-id flow.
func fetchStagedFor(c *api.Client, kind string, id int64) (*stagedElement, error) {
	ctx := context.Background()
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
