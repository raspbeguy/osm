package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type readerLoadedMsg struct {
	msgID int64
	msg   *api.Message
	err   error
}

type readerDeletedMsg struct {
	msgID int64
	err   error
}

type readerModel struct {
	client     *api.Client
	parent     screen
	msgID      int64
	spinner    spinner.Model
	viewport   viewport.Model
	msg        *api.Message
	err        error
	loading    bool
	confirming bool
	deleting   bool
}

func newReader(c *api.Client) readerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return readerModel{
		client:   c,
		spinner:  s,
		viewport: viewport.New(60, 15),
	}
}

func (m readerModel) Init() tea.Cmd { return nil }

func (m readerModel) show(msgID int64, parent screen) (readerModel, tea.Cmd) {
	m.parent = parent
	m.msgID = msgID
	m.loading = true
	m.msg = nil
	m.err = nil
	m.confirming = false
	m.deleting = false
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m readerModel) load() tea.Cmd {
	client := m.client
	id := m.msgID
	return func() tea.Msg {
		msg, err := client.GetMessage(context.Background(), id)
		return readerLoadedMsg{msgID: id, msg: msg, err: err}
	}
}

func (m readerModel) deleteAction() tea.Cmd {
	client := m.client
	id := m.msgID
	return func() tea.Msg {
		err := client.DeleteMessage(context.Background(), id)
		return readerDeletedMsg{msgID: id, err: err}
	}
}

func (m readerModel) Update(msg tea.Msg) (readerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case readerLoadedMsg:
		if msg.msgID != m.msgID {
			return m, nil
		}
		m.loading = false
		m.msg = msg.msg
		m.err = msg.err
		m = m.rewrap()
		return m, nil
	case readerDeletedMsg:
		if msg.msgID != m.msgID {
			return m, nil
		}
		m.deleting = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		parent := m.parent
		return m, func() tea.Msg {
			return navigateMsg{to: parent, refresh: true}
		}
	case spinner.TickMsg:
		if !m.loading && !m.deleting {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if m.confirming {
			switch msg.String() {
			case "y", "Y":
				m.confirming = false
				m.deleting = true
				return m, tea.Batch(m.spinner.Tick, m.deleteAction())
			case "n", "N", "esc":
				m.confirming = false
				return m, nil
			}
			return m, nil
		}
		if msg.String() == "d" && m.msg != nil {
			m.confirming = true
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m readerModel) rewrap() readerModel {
	if m.msg != nil {
		m.viewport.SetContent(wrapText(m.msg.Body, m.viewport.Width))
	}
	return m
}

func (m readerModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading message..."
	}
	if m.deleting {
		return m.spinner.View() + " deleting..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc to go back")
	}
	if m.msg == nil {
		return "no message\n" + footerStyle.Render("esc to go back")
	}
	header := headerStyle.Render(m.msg.Title) + "\n" +
		mutedStyle.Render(fmt.Sprintf("from %s • to %s • %s", m.msg.FromUser, m.msg.ToUser, m.msg.SentOn))
	footer := "esc back, d delete"
	if m.confirming {
		footer = errorStyle.Render("delete this message? y/n")
	} else {
		footer = footerStyle.Render(footer)
	}
	return header + "\n\n" + m.viewport.View() + "\n" + footer
}
