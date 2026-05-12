package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type screen int

const (
	screenMenu screen = iota
	screenProfile
	screenInbox
	screenOutbox
	screenChangesets
	screenChangesetView
	screenNotes
	screenDoctor
	screenHistory
	screenTraces
	screenComposeChangeset
	screenAddElement
	screenEditElement
	screenEditMembers
	screenSubmitChangeset
)

// navigateMsg requests a screen change. refresh asks the destination to
// re-load its contents. itemID is the per-screen target id (changeset id for
// changeset view, element id for history view, etc). kind (node|way|relation)
// selects history's element kind on direct-load.
type navigateMsg struct {
	to      screen
	itemID  int64
	kind    string
	parent  screen
	refresh bool
}

type rootModel struct {
	client      *api.Client
	width       int
	height      int
	screen      screen
	menu        menuModel
	profile     profileModel
	inbox       messagesModel
	outbox      messagesModel
	changesets  changesetsModel
	csview      changesetViewModel
	notes       notesModel
	doctor      doctorModel
	history     historyModel
	traces      tracesModel
	compose     composeChangesetModel
	addElement  addElementModel
	editEl      editElementModel
	editMembers editMembersModel
	submit      submitChangesetModel
}

func newRoot(c *api.Client) rootModel {
	return rootModel{
		client:      c,
		screen:      screenMenu,
		menu:        newMenu(),
		profile:     newProfile(c),
		inbox:       newMessages(c, dirInbox),
		outbox:      newMessages(c, dirOutbox),
		changesets:  newChangesets(c),
		csview:      newChangesetView(c),
		notes:       newNotes(c),
		doctor:      newDoctor(c),
		history:     newHistory(c),
		traces:      newTraces(c),
		compose:     newCompose(c),
		addElement:  newAddElement(c),
		editEl:      newEditElement(),
		editMembers: newEditMembers(),
		submit:      newSubmit(c),
	}
}

func (m rootModel) Init() tea.Cmd { return nil }

// splitWidths returns (leftPaneInnerWidth, rightPaneInnerWidth) for a 50/50
// split given a terminal width. Accounts for 2 borders of 1 char each per pane.
func splitWidths(total int) (int, int) {
	usable := total - 4
	if usable < 8 {
		usable = 8
	}
	l := usable / 2
	return l, usable - l
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		paneH := msg.Height - 3
		if paneH < 5 {
			paneH = 5
		}
		leftW, rightW := splitWidths(msg.Width)

		m.menu.list.SetSize(msg.Width, msg.Height-2)
		m.profile.viewport.Width = msg.Width
		m.profile.viewport.Height = msg.Height - 4

		m.inbox.list.SetSize(leftW, paneH)
		m.inbox.viewport.Width = rightW
		m.inbox.viewport.Height = paneH
		m.outbox.list.SetSize(leftW, paneH)
		m.outbox.viewport.Width = rightW
		m.outbox.viewport.Height = paneH

		m.changesets.list.SetSize(msg.Width, msg.Height-3)
		csH := msg.Height - 8
		if csH < 5 {
			csH = 5
		}
		m.csview.viewport.Width = msg.Width
		m.csview.viewport.Height = csH
		m.csview.elementsList.SetSize(leftW, csH)
		m.csview.detailViewport.Width = rightW
		m.csview.detailViewport.Height = csH

		m.notes.viewport.Width = msg.Width
		m.notes.viewport.Height = msg.Height - 6
		m.notes.list.SetSize(msg.Width, msg.Height-3)
		m.notes.input.Width = msg.Width - 4

		m.doctor.viewport.Width = msg.Width
		m.doctor.viewport.Height = msg.Height - 2

		m.history.viewport.Width = msg.Width
		m.history.viewport.Height = msg.Height - 6
		m.history.input.Width = msg.Width - 4

		m.traces.list.SetSize(leftW, paneH)
		m.traces.viewport.Width = rightW
		m.traces.viewport.Height = paneH

		m.compose.list.SetSize(leftW, paneH)
		m.compose.viewport.Width = rightW
		m.compose.viewport.Height = paneH
		m.addElement.input.Width = msg.Width - 4
		m.editEl.list.SetSize(msg.Width, msg.Height-5)
		m.editEl.input.Width = msg.Width - 4
		m.editMembers.list.SetSize(msg.Width, msg.Height-5)
		m.editMembers.input.Width = msg.Width - 4
		m.submit.commentInput.Width = msg.Width - 4
		m.submit.tagInput.Width = msg.Width - 4
		m.submit.tagsList.SetSize(msg.Width, 8)

		m.profile = m.profile.rewrap()
		m.inbox = m.inbox.rewrap()
		m.outbox = m.outbox.rewrap()
		m.csview = m.csview.rewrap()
		m.notes = m.notes.rewrap()
		m.history = m.history.rewrap()
		m.traces = m.traces.rewrap()
		m.compose = m.compose.rewrap()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			switch m.screen {
			case screenMenu:
				return m, tea.Quit
			case screenInbox, screenOutbox:
				if m.activeMessages().confirming {
					break
				}
				m.screen = screenMenu
				return m, nil
			case screenChangesets:
				if m.changesets.list.FilterState() != list.Unfiltered {
					break // let the list cancel/clear the filter
				}
				m.screen = screenMenu
				return m, nil
			case screenChangesetView:
				m.screen = screenChangesets
				return m, nil
			case screenAddElement, screenEditElement, screenSubmitChangeset:
				m.screen = screenComposeChangeset
				return m, nil
			case screenEditMembers:
				m.screen = screenEditElement
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
	case stagedAddedMsg:
		if msg.elem != nil {
			m.compose.staged = append(m.compose.staged, msg.elem)
			m.compose.refreshList()
			m.compose.list.Select(len(m.compose.staged) - 1)
			m.compose = m.compose.rewrap()
		}
		m.screen = screenComposeChangeset
		return m, nil
	case submitConfirmedMsg:
		m.compose.staged = nil
		m.compose.refreshList()
		m.screen = screenMenu
		return m, nil
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
	case screenChangesets:
		m.changesets, cmd = m.changesets.Update(msg)
	case screenChangesetView:
		m.csview, cmd = m.csview.Update(msg)
	case screenNotes:
		m.notes, cmd = m.notes.Update(msg)
	case screenDoctor:
		m.doctor, cmd = m.doctor.Update(msg)
	case screenHistory:
		m.history, cmd = m.history.Update(msg)
	case screenTraces:
		m.traces, cmd = m.traces.Update(msg)
	case screenComposeChangeset:
		m.compose, cmd = m.compose.Update(msg)
	case screenAddElement:
		m.addElement, cmd = m.addElement.Update(msg)
	case screenEditElement:
		m.editEl, cmd = m.editEl.Update(msg)
	case screenEditMembers:
		m.editMembers, cmd = m.editMembers.Update(msg)
	case screenSubmitChangeset:
		m.submit, cmd = m.submit.Update(msg)
	}
	return m, cmd
}

func (m rootModel) activeMessages() messagesModel {
	if m.screen == screenOutbox {
		return m.outbox
	}
	return m.inbox
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
	case screenDoctor:
		var cmd tea.Cmd
		m.doctor, cmd = m.doctor.show()
		return m, cmd
	case screenHistory:
		var cmd tea.Cmd
		if msg.kind != "" {
			m.history, cmd = m.history.showResult(msg.kind, msg.itemID)
		} else {
			m.history, cmd = m.history.show()
		}
		return m, cmd
	case screenTraces:
		if msg.refresh || (len(m.traces.list.Items()) == 0 && !m.traces.loading) {
			var cmd tea.Cmd
			m.traces, cmd = m.traces.show()
			return m, cmd
		}
		return m, nil
	case screenComposeChangeset:
		var cmd tea.Cmd
		m.compose, cmd = m.compose.show()
		return m, cmd
	case screenAddElement:
		var cmd tea.Cmd
		m.addElement, cmd = m.addElement.show()
		return m, cmd
	case screenEditElement:
		if sel := m.compose.selectedStaged(); sel != nil {
			m.editEl = m.editEl.show(sel)
		}
		return m, nil
	case screenEditMembers:
		if sel := m.compose.selectedStaged(); sel != nil {
			m.editMembers = m.editMembers.show(sel)
		}
		return m, nil
	case screenSubmitChangeset:
		var cmd tea.Cmd
		m.submit, cmd = m.submit.show(m.compose.staged)
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
	case screenChangesets:
		return m.changesets.View()
	case screenChangesetView:
		return m.csview.View()
	case screenNotes:
		return m.notes.View()
	case screenDoctor:
		return m.doctor.View()
	case screenHistory:
		return m.history.View()
	case screenTraces:
		return m.traces.View()
	case screenComposeChangeset:
		return m.compose.View()
	case screenAddElement:
		return m.addElement.View()
	case screenEditElement:
		return m.editEl.View()
	case screenEditMembers:
		return m.editMembers.View()
	case screenSubmitChangeset:
		return m.submit.View()
	}
	return ""
}
