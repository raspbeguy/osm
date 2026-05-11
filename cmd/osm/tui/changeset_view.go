package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
)

type changesetViewLoadedMsg struct {
	csID osm.ChangesetID
	cs   *osm.Changeset
	err  error
}

type changesetViewModel struct {
	client   *api.Client
	csID     osm.ChangesetID
	spinner  spinner.Model
	viewport viewport.Model
	cs       *osm.Changeset
	err      error
	loading  bool
}

func newChangesetView(c *api.Client) changesetViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return changesetViewModel{
		client:   c,
		spinner:  s,
		viewport: viewport.New(60, 15),
	}
}

func (m changesetViewModel) Init() tea.Cmd { return nil }

func (m changesetViewModel) show(id int64) (changesetViewModel, tea.Cmd) {
	m.csID = osm.ChangesetID(id)
	m.loading = true
	m.err = nil
	m.cs = nil
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m changesetViewModel) load() tea.Cmd {
	client := m.client
	id := m.csID
	return func() tea.Msg {
		cs, err := client.GetChangeset(context.Background(), id)
		return changesetViewLoadedMsg{csID: id, cs: cs, err: err}
	}
}

func (m changesetViewModel) Update(msg tea.Msg) (changesetViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case changesetViewLoadedMsg:
		if msg.csID != m.csID {
			return m, nil
		}
		m.loading = false
		m.cs = msg.cs
		m.err = msg.err
		if m.cs != nil {
			m.viewport.SetContent(formatChangesetBody(m.cs))
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

func (m changesetViewModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading changeset..."
	}
	if m.err != nil {
		return "error: " + m.err.Error() + "\n\nesc back"
	}
	if m.cs == nil {
		return "no changeset\n\nesc back"
	}
	closedAt := "(open)"
	if !m.cs.ClosedAt.IsZero() {
		closedAt = m.cs.ClosedAt.Format(time.RFC3339)
	}
	header := fmt.Sprintf("ID: %d\nUser: %s\nOpen: %v\nCreated: %s\nClosed: %s",
		m.cs.ID, m.cs.User, m.cs.Open, m.cs.CreatedAt.Format(time.RFC3339), closedAt)
	return header + "\n\n" + m.viewport.View() + "\n\nesc back"
}

func formatChangesetBody(cs *osm.Changeset) string {
	var sb strings.Builder
	sb.WriteString("Tags:\n")
	for _, t := range cs.Tags {
		fmt.Fprintf(&sb, "  %s = %s\n", t.Key, t.Value)
	}
	if cs.Discussion != nil && len(cs.Discussion.Comments) > 0 {
		sb.WriteString("\nComments:\n")
		for _, c := range cs.Discussion.Comments {
			fmt.Fprintf(&sb, "\n%s by %s:\n%s\n", c.Timestamp.Format(time.RFC3339), c.User, c.Text)
		}
	}
	return sb.String()
}
