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
	screenOutbox
	screenReader
	screenChangesets
	screenChangesetView
	screenNotes
)

// navigateMsg requests a screen change. refresh asks the destination to
// re-load its contents. msgID and parent are used by screenReader.
type navigateMsg struct {
	to      screen
	itemID  int64
	parent  screen
	refresh bool
}

type rootModel struct {
	client     *api.Client
	width      int
	height     int
	screen     screen
	menu       menuModel
	profile    profileModel
	inbox      messagesModel
	outbox     messagesModel
	reader     readerModel
	changesets changesetsModel
	csview     changesetViewModel
	notes      notesModel
}

func newRoot(c *api.Client) rootModel {
	return rootModel{
		client:     c,
		screen:     screenMenu,
		menu:       newMenu(),
		profile:    newProfile(c),
		inbox:      newMessages(c, dirInbox),
		outbox:     newMessages(c, dirOutbox),
		reader:     newReader(c),
		changesets: newChangesets(c),
		csview:     newChangesetView(c),
		notes:      newNotes(c),
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
		m.outbox.list.SetSize(msg.Width, msg.Height-3)
		m.reader.viewport.Width = msg.Width
		m.reader.viewport.Height = msg.Height - 8
		m.changesets.list.SetSize(msg.Width, msg.Height-3)
		m.csview.viewport.Width = msg.Width
		m.csview.viewport.Height = msg.Height - 8
		m.notes.viewport.Width = msg.Width
		m.notes.viewport.Height = msg.Height - 6
		m.notes.list.SetSize(msg.Width, msg.Height-3)
		m.notes.input.Width = msg.Width - 4
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.screen {
			case screenMenu:
				return m, tea.Quit
			case screenReader:
				if m.reader.confirming {
					break // let reader handle the cancel
				}
				dest := m.reader.parent
				if dest == 0 {
					dest = screenMenu
				}
				m.screen = dest
				return m, nil
			case screenChangesetView:
				m.screen = screenChangesets
				return m, nil
			default:
				m.screen = screenMenu
				return m, nil
			}
		case "q":
			if m.screen == screenMenu {
				return m, tea.Quit
			}
		}
	case navigateMsg:
		return m.handleNavigate(msg)
	}

	var cmd tea.Cmd
	switch m.screen {
	case screenMenu:
		m.menu, cmd = m.menu.Update(msg)
	case screenProfile:
		m.profile, cmd = m.profile.Update(msg)
	case screenInbox:
		m.inbox, cmd = m.inbox.Update(msg)
	case screenOutbox:
		m.outbox, cmd = m.outbox.Update(msg)
	case screenReader:
		m.reader, cmd = m.reader.Update(msg)
	case screenChangesets:
		m.changesets, cmd = m.changesets.Update(msg)
	case screenChangesetView:
		m.csview, cmd = m.csview.Update(msg)
	case screenNotes:
		m.notes, cmd = m.notes.Update(msg)
	}
	return m, cmd
}

func (m rootModel) handleNavigate(msg navigateMsg) (rootModel, tea.Cmd) {
	m.screen = msg.to
	switch msg.to {
	case screenProfile:
		var cmd tea.Cmd
		m.profile, cmd = m.profile.show()
		return m, cmd
	case screenInbox:
		if msg.refresh || (len(m.inbox.list.Items()) == 0 && !m.inbox.loading) {
			var cmd tea.Cmd
			m.inbox, cmd = m.inbox.show()
			return m, cmd
		}
		return m, nil
	case screenOutbox:
		if msg.refresh || (len(m.outbox.list.Items()) == 0 && !m.outbox.loading) {
			var cmd tea.Cmd
			m.outbox, cmd = m.outbox.show()
			return m, cmd
		}
		return m, nil
	case screenReader:
		var cmd tea.Cmd
		m.reader, cmd = m.reader.show(msg.itemID, msg.parent)
		return m, cmd
	case screenChangesets:
		if msg.refresh || (len(m.changesets.list.Items()) == 0 && !m.changesets.loading) {
			var cmd tea.Cmd
			m.changesets, cmd = m.changesets.show()
			return m, cmd
		}
		return m, nil
	case screenChangesetView:
		var cmd tea.Cmd
		m.csview, cmd = m.csview.show(msg.itemID)
		return m, cmd
	case screenNotes:
		var cmd tea.Cmd
		m.notes, cmd = m.notes.show()
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
	case screenOutbox:
		return m.outbox.View()
	case screenReader:
		return m.reader.View()
	case screenChangesets:
		return m.changesets.View()
	case screenChangesetView:
		return m.csview.View()
	case screenNotes:
		return m.notes.View()
	}
	return ""
}
