package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type screen int

const (
	screenMenu screen = iota
	screenProfile
	screenInbox
)

type rootModel struct {
	client  *api.Client
	width   int
	height  int
	screen  screen
	menu    menuModel
	profile profileModel
	inbox   messagesModel
}

func newRoot(c *api.Client) rootModel {
	return rootModel{
		client:  c,
		screen:  screenMenu,
		menu:    newMenu(),
		profile: newProfile(c),
		inbox:   newMessages(c, dirInbox),
	}
}

func (m rootModel) Init() tea.Cmd { return nil }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.menu.list.SetSize(msg.Width, msg.Height-2)
		m.profile.viewport.Width = msg.Width
		m.profile.viewport.Height = msg.Height - 4
		m.inbox.list.SetSize(msg.Width, msg.Height-3)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.screen != screenMenu {
				m.screen = screenMenu
				return m, nil
			}
			return m, tea.Quit
		case "q":
			if m.screen == screenMenu {
				return m, tea.Quit
			}
		case "enter":
			if m.screen == screenMenu {
				if target, ok := m.menu.selected(); ok {
					return m.navigate(target)
				}
				return m, nil
			}
		}
	}
	var cmd tea.Cmd
	switch m.screen {
	case screenMenu:
		m.menu, cmd = m.menu.Update(msg)
	case screenProfile:
		m.profile, cmd = m.profile.Update(msg)
	case screenInbox:
		m.inbox, cmd = m.inbox.Update(msg)
	}
	return m, cmd
}

func (m rootModel) navigate(target screen) (rootModel, tea.Cmd) {
	m.screen = target
	switch target {
	case screenProfile:
		var cmd tea.Cmd
		m.profile, cmd = m.profile.show()
		return m, cmd
	case screenInbox:
		var cmd tea.Cmd
		m.inbox, cmd = m.inbox.show()
		return m, cmd
	}
	return m, nil
}

func (m rootModel) View() string {
	switch m.screen {
	case screenMenu:
		return m.menu.View()
	case screenProfile:
		return m.profile.View()
	case screenInbox:
		return m.inbox.View()
	}
	return ""
}
