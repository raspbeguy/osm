package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

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

type messageItem struct {
	msg *api.Message
}

func (i messageItem) Title() string {
	who := i.msg.FromUser
	if who == "" {
		who = i.msg.ToUser
	}
	return fmt.Sprintf("%s  %s", who, i.msg.Title)
}

func (i messageItem) Description() string {
	if len(i.msg.SentOn) >= 10 {
		return i.msg.SentOn[:10]
	}
	return i.msg.SentOn
}

func (i messageItem) FilterValue() string { return i.msg.Title }

type messagesModel struct {
	client    *api.Client
	direction messagesDirection
	spinner   spinner.Model
	list      list.Model
	err       error
	loading   bool
}

func newMessages(c *api.Client, dir messagesDirection) messagesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, list.NewDefaultDelegate(), 60, 20)
	l.Title = dir.String()
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return messagesModel{client: c, direction: dir, spinner: s, list: l}
}

func (m messagesModel) Init() tea.Cmd { return nil }

func (m messagesModel) show() (messagesModel, tea.Cmd) {
	m.loading = true
	m.err = nil
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
			msgs, err = client.ListInbox(context.Background())
		} else {
			msgs, err = client.ListOutbox(context.Background())
		}
		return messagesLoadedMsg{direction: dir, messages: msgs, err: err}
	}
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
				parent := screenInbox
				return m, func() tea.Msg {
					return navigateMsg{to: screenReader, msgID: sel.ID, parent: parent}
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m messagesModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading " + m.direction.String() + "..."
	}
	if m.err != nil {
		return "error: " + m.err.Error() + "\n\nesc to go back, r to retry"
	}
	return m.list.View() + "\nesc back, r refresh"
}

func (m messagesModel) selected() *api.Message {
	if i, ok := m.list.SelectedItem().(messageItem); ok {
		return i.msg
	}
	return nil
}
