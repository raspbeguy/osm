package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	name   string
	desc   string
	target screen
}

func (i menuItem) Title() string       { return i.name }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.name }

type menuModel struct {
	list list.Model
}

func newMenu() menuModel {
	items := []list.Item{
		menuItem{name: "Profile", desc: "your osm account", target: screenProfile},
		menuItem{name: "Inbox", desc: "received messages", target: screenInbox},
		menuItem{name: "Outbox", desc: "sent messages", target: screenOutbox},
		menuItem{name: "Changesets", desc: "your changesets", target: screenChangesets},
		menuItem{name: "Traces", desc: "your gps traces", target: screenTraces},
		menuItem{name: "Notes", desc: "lookup a note by id or bbox", target: screenNotes},
		menuItem{name: "History", desc: "version history of an element", target: screenHistory},
		menuItem{name: "Server info", desc: "capabilities and permissions", target: screenDoctor},
	}
	l := list.New(items, list.NewDefaultDelegate(), 40, 20)
	l.Title = "osm tui"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return menuModel{list: l}
}

func (m menuModel) Init() tea.Cmd { return nil }

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "enter" {
		if target, ok := m.selected(); ok {
			return m, func() tea.Msg {
				return navigateMsg{to: target, refresh: true}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m menuModel) View() string {
	return m.list.View()
}

func (m menuModel) selected() (screen, bool) {
	if i, ok := m.list.SelectedItem().(menuItem); ok {
		return i.target, true
	}
	return 0, false
}
