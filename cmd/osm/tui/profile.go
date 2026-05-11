package tui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type profileLoadedMsg struct {
	user *api.User
	err  error
}

type profileModel struct {
	client   *api.Client
	spinner  spinner.Model
	viewport viewport.Model
	user     *api.User
	err      error
	loading  bool
}

func newProfile(c *api.Client) profileModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return profileModel{
		client:   c,
		spinner:  s,
		viewport: viewport.New(60, 20),
	}
}

func (m profileModel) Init() tea.Cmd { return nil }

func (m profileModel) show() (profileModel, tea.Cmd) {
	m.loading = true
	m.user = nil
	m.err = nil
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m profileModel) load() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		u, err := client.Whoami(context.Background())
		return profileLoadedMsg{user: u, err: err}
	}
}

func (m profileModel) Update(msg tea.Msg) (profileModel, tea.Cmd) {
	switch msg := msg.(type) {
	case profileLoadedMsg:
		m.loading = false
		m.user = msg.user
		m.err = msg.err
		if m.user != nil {
			m.viewport.SetContent(formatUser(m.user))
		}
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m profileModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading profile..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc to go back")
	}
	return m.viewport.View() + "\n" + footerStyle.Render("esc to go back")
}

func formatUser(u *api.User) string {
	return fmt.Sprintf("Display name : %s\nID           : %d\nCreated      : %s\nChangesets   : %d\nLanguages    : %v\n\n%s",
		u.DisplayName, u.ID, u.AccountCreated, u.ChangesetCount, u.Languages, u.Description)
}
