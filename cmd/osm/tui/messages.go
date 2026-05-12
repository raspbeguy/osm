package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raspbeguy/osm/api"
)

type messagesDirection int

const (
	dirInbox messagesDirection = iota
	dirOutbox
)

func (d messagesDirection) String() string {
	if d == dirOutbox {
		return "Outbox"
	}
	return "Inbox"
}

type messagesLoadedMsg struct {
	direction messagesDirection
	messages  []*api.Message
	err       error
}

type messageBodyLoadedMsg struct {
	direction messagesDirection
	id        int64
	msg       *api.Message
	err       error
}

type messageDeletedMsg struct {
	direction messagesDirection
	id        int64
	err       error
}

type messageItem struct {
	msg *api.Message
}

func (i messageItem) Title() string {
	who := i.msg.FromUser
	if who == "" {
		who = i.msg.ToUser
	}
	date := i.msg.SentOn
	if len(date) >= 10 {
		date = date[:10]
	}
	return fmt.Sprintf("%s  %s  %s", date, who, i.msg.Title)
}

func (i messageItem) Description() string { return "" }

func (i messageItem) FilterValue() string { return i.Title() }

type messagesModel struct {
	client      *api.Client
	direction   messagesDirection
	spinner     spinner.Model
	list        list.Model
	viewport    viewport.Model
	bodies      map[int64]*api.Message
	bodyLoading map[int64]bool
	err         error
	loading     bool
	deleting    bool
	confirming  bool
	focus       int // 0=list, 1=body
	lastID      int64
}

func newMessages(c *api.Client, dir messagesDirection) messagesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, newCompactDelegate(), 40, 20)
	l.Title = dir.String()
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return messagesModel{
		client:      c,
		direction:   dir,
		spinner:     s,
		list:        l,
		viewport:    viewport.New(40, 20),
		bodies:      map[int64]*api.Message{},
		bodyLoading: map[int64]bool{},
	}
}

func (m messagesModel) Init() tea.Cmd { return nil }

func (m messagesModel) show() (messagesModel, tea.Cmd) {
	m.loading = true
	m.err = nil
	m.confirming = false
	m.deleting = false
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m messagesModel) load() tea.Cmd {
	client := m.client
	dir := m.direction
	return func() tea.Msg {
		var (
			msgs []*api.Message
			err  error
		)
		if dir == dirInbox {
			msgs, err = client.ListInbox(programCtx)
		} else {
			msgs, err = client.ListOutbox(programCtx)
		}
		return messagesLoadedMsg{direction: dir, messages: msgs, err: err}
	}
}

func (m messagesModel) fetchBody(id int64) tea.Cmd {
	client := m.client
	dir := m.direction
	return func() tea.Msg {
		msg, err := client.GetMessage(programCtx, id)
		return messageBodyLoadedMsg{direction: dir, id: id, msg: msg, err: err}
	}
}

func (m messagesModel) deleteAction(id int64) tea.Cmd {
	client := m.client
	dir := m.direction
	return func() tea.Msg {
		err := client.DeleteMessage(programCtx, id)
		return messageDeletedMsg{direction: dir, id: id, err: err}
	}
}

func (m messagesModel) currentID() int64 {
	if i, ok := m.list.SelectedItem().(messageItem); ok {
		return i.msg.ID
	}
	return 0
}

func (m messagesModel) ensureBody(id int64) tea.Cmd {
	if id == 0 {
		return nil
	}
	if _, ok := m.bodies[id]; ok {
		return nil
	}
	if m.bodyLoading[id] {
		return nil
	}
	m.bodyLoading[id] = true
	return tea.Batch(m.spinner.Tick, m.fetchBody(id))
}

func (m messagesModel) Update(msg tea.Msg) (messagesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messagesLoadedMsg:
		if msg.direction != m.direction {
			return m, nil
		}
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, len(msg.messages))
			for i, x := range msg.messages {
				items[i] = messageItem{msg: x}
			}
			m.list.SetItems(items)
		}
		id := m.currentID()
		m.lastID = id
		cmd := m.ensureBody(id)
		m = m.rewrap()
		return m, cmd
	case messageBodyLoadedMsg:
		if msg.direction != m.direction {
			return m, nil
		}
		delete(m.bodyLoading, msg.id)
		var cmd tea.Cmd
		if msg.err == nil && msg.msg != nil {
			m.bodies[msg.id] = msg.msg
			// Auto-mark inbox messages as read on first body fetch.
			if m.direction == dirInbox && !msg.msg.Read {
				client := m.client
				id := msg.id
				cmd = func() tea.Msg {
					_ = client.MarkRead(programCtx, id, true)
					return nil
				}
			}
		}
		m = m.rewrap()
		return m, cmd
	case messageDeletedMsg:
		if msg.direction != m.direction {
			return m, nil
		}
		m.deleting = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := m.list.Items()
		for i, it := range items {
			if mi, ok := it.(messageItem); ok && mi.msg.ID == msg.id {
				items = append(items[:i], items[i+1:]...)
				m.list.SetItems(items)
				break
			}
		}
		delete(m.bodies, msg.id)
		m = m.rewrap()
		return m, nil
	case spinner.TickMsg:
		if !m.loading && !m.deleting && len(m.bodyLoading) == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				id := m.currentID()
				m.confirming = false
				if id != 0 {
					m.deleting = true
					return m, tea.Batch(m.spinner.Tick, m.deleteAction(id))
				}
			case "n", "N", "esc":
				m.confirming = false
			}
			return m, nil
		}
		if !inFilter(m.list) {
			switch msg.String() {
			case "r":
				return m.show()
			case "tab":
				m.focus = 1 - m.focus
				return m, nil
			case "d":
				if m.direction == dirInbox && m.currentID() != 0 {
					m.confirming = true
					return m, nil
				}
			}
		}
	}

	prevID := m.lastID
	var cmd tea.Cmd
	if m.focus == 0 {
		m.list, cmd = m.list.Update(msg)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	curID := m.currentID()
	m.lastID = curID
	var fetchCmd tea.Cmd
	if curID != 0 && curID != prevID {
		fetchCmd = m.ensureBody(curID)
		m = m.rewrap()
	}
	return m, tea.Batch(cmd, fetchCmd)
}

func (m messagesModel) rewrap() messagesModel {
	id := m.currentID()
	if id == 0 {
		m.viewport.SetContent(mutedStyle.Render("(no selection)"))
		return m
	}
	if m.bodyLoading[id] {
		m.viewport.SetContent(m.spinner.View() + " loading...")
		return m
	}
	msg, ok := m.bodies[id]
	if !ok {
		m.viewport.SetContent(mutedStyle.Render("(no body)"))
		return m
	}
	header := headerStyle.Render(msg.Title) + "\n" +
		mutedStyle.Render(fmt.Sprintf("from %s • to %s • %s", msg.FromUser, msg.ToUser, msg.SentOn))
	var rendered string
	if msg.BodyFormat == "markdown" || msg.BodyFormat == "" {
		rendered = renderMarkdown(msg.Body, m.viewport.Width)
	} else {
		rendered = wrapText(msg.Body, m.viewport.Width)
	}
	m.viewport.SetContent(header + "\n\n" + rendered)
	return m
}

func (m messagesModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading " + m.direction.String() + "..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back, r retry")
	}
	if len(m.list.Items()) == 0 {
		return mutedStyle.Render("(no messages)") + "\n" + footerStyle.Render("esc back, r refresh")
	}
	leftStyle, rightStyle := paneFocused, paneUnfocused
	if m.focus == 1 {
		leftStyle, rightStyle = paneUnfocused, paneFocused
	}
	left := leftStyle.Render(m.list.View())
	right := rightStyle.Render(m.viewport.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := "esc back, tab swap pane, r refresh"
	if m.direction == dirInbox {
		footer += ", d delete"
	}
	if m.confirming {
		footer = errorStyle.Render("delete this message? y/n")
	}
	if m.deleting {
		footer = m.spinner.View() + " deleting..."
	}
	return body + "\n" + footerStyle.Render(footer)
}
