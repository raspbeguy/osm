package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type traceItem struct{ t *api.Trace }

func (i traceItem) Title() string {
	desc := i.t.Description
	if desc == "" {
		desc = i.t.Name
	}
	return fmt.Sprintf("%d  %s", i.t.ID, desc)
}

func (i traceItem) Description() string {
	date := i.t.Timestamp
	if len(date) >= 10 {
		date = date[:10]
	}
	return fmt.Sprintf("%s  [%s]", date, i.t.Visibility)
}

func (i traceItem) FilterValue() string { return i.t.Description }

type tracesLoadedMsg struct {
	traces []*api.Trace
	err    error
}

type tracesModel struct {
	client  *api.Client
	spinner spinner.Model
	list    list.Model
	err     error
	loading bool
}

func newTraces(c *api.Client) tracesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, list.NewDefaultDelegate(), 60, 20)
	l.Title = "Traces"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return tracesModel{client: c, spinner: s, list: l}
}

func (m tracesModel) Init() tea.Cmd { return nil }

func (m tracesModel) show() (tracesModel, tea.Cmd) {
	m.loading = true
	m.err = nil
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m tracesModel) load() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ts, err := client.ListTraces(context.Background())
		return tracesLoadedMsg{traces: ts, err: err}
	}
}

func (m tracesModel) Update(msg tea.Msg) (tracesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tracesLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, len(msg.traces))
			for i, t := range msg.traces {
				items[i] = traceItem{t: t}
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
		switch msg.String() {
		case "r":
			return m.show()
		case "enter":
			if sel := m.selected(); sel != nil {
				return m, func() tea.Msg {
					return navigateMsg{to: screenTraceView, itemID: sel.ID, parent: screenTraces}
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m tracesModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading traces..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back, r retry")
	}
	return m.list.View() + "\n" + footerStyle.Render("esc back, enter open, r refresh")
}

func (m tracesModel) selected() *api.Trace {
	if i, ok := m.list.SelectedItem().(traceItem); ok {
		return i.t
	}
	return nil
}
