package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
)

type changesetItem struct {
	cs *osm.Changeset
}

func (i changesetItem) Title() string {
	c := i.cs.Comment()
	if c == "" {
		c = "(no comment)"
	}
	return fmt.Sprintf("%s  %d  %s  %s", i.cs.CreatedAt.Format("2006-01-02"), i.cs.ID, i.cs.User, c)
}

func (i changesetItem) Description() string { return "" }

func (i changesetItem) FilterValue() string { return i.Title() }

type changesetsLoadedMsg struct {
	changesets []*osm.Changeset
	err        error
}

type changesetsModel struct {
	client  *api.Client
	spinner spinner.Model
	list    list.Model
	err     error
	loading bool
}

func newChangesets(c *api.Client) changesetsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, newCompactDelegate(), 60, 20)
	l.Title = "Your changesets"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return changesetsModel{client: c, spinner: s, list: l}
}

func (m changesetsModel) Init() tea.Cmd { return nil }

func (m changesetsModel) show() (changesetsModel, tea.Cmd) {
	m.loading = true
	m.err = nil
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m changesetsModel) load() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		u, err := client.Whoami(context.Background())
		if err != nil {
			return changesetsLoadedMsg{err: err}
		}
		css, err := client.ListChangesets(context.Background(), api.ChangesetFilter{UserID: u.ID})
		return changesetsLoadedMsg{changesets: css, err: err}
	}
}

func (m changesetsModel) Update(msg tea.Msg) (changesetsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case changesetsLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, len(msg.changesets))
			for i, x := range msg.changesets {
				items[i] = changesetItem{cs: x}
			}
			m.list.SetItems(items)
		}
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "r":
				return m.show()
			case "enter":
				if sel := m.selected(); sel != nil {
					return m, func() tea.Msg {
						return navigateMsg{to: screenChangesetView, itemID: int64(sel.ID)}
					}
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m changesetsModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading changesets..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back, r retry")
	}
	return m.list.View() + "\n" + footerStyle.Render("esc back, enter open, / filter, r refresh")
}

func (m changesetsModel) selected() *osm.Changeset {
	if i, ok := m.list.SelectedItem().(changesetItem); ok {
		return i.cs
	}
	return nil
}
