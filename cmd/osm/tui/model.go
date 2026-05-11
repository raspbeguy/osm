package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type rootModel struct {
	client *api.Client
}

func newRoot(c *api.Client) rootModel {
	return rootModel{client: c}
}

func (m rootModel) Init() tea.Cmd { return nil }

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m rootModel) View() string {
	return "osm tui\n\npress q to quit\n"
}
