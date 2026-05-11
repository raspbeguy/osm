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

type changesetXMLLoadedMsg struct {
	csID osm.ChangesetID
	xml  string
	err  error
}

type changesetViewModel struct {
	client     *api.Client
	csID       osm.ChangesetID
	spinner    spinner.Model
	viewport   viewport.Model
	cs         *osm.Changeset
	err        error
	loading    bool
	xml        string
	xmlLoading bool
	showXML    bool
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
	m.xml = ""
	m.xmlLoading = false
	m.showXML = false
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m changesetViewModel) loadXML() tea.Cmd {
	client := m.client
	id := m.csID
	return func() tea.Msg {
		xml, err := client.DownloadChangeset(context.Background(), id)
		return changesetXMLLoadedMsg{csID: id, xml: xml, err: err}
	}
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
		m = m.rewrap()
		return m, nil
	case changesetXMLLoadedMsg:
		if msg.csID != m.csID {
			return m, nil
		}
		m.xmlLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.xml = msg.xml
		if m.showXML {
			m.viewport.SetContent(wrapText(m.xml, m.viewport.Width))
		}
		return m, nil
	case spinner.TickMsg:
		if !m.loading && !m.xmlLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "x" && m.cs != nil {
			m.showXML = !m.showXML
			m = m.rewrap()
			if m.showXML && m.xml == "" && !m.xmlLoading {
				m.xmlLoading = true
				return m, tea.Batch(m.spinner.Tick, m.loadXML())
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m changesetViewModel) rewrap() changesetViewModel {
	if m.showXML && m.xml != "" {
		m.viewport.SetContent(wrapText(m.xml, m.viewport.Width))
	} else if m.cs != nil {
		m.viewport.SetContent(wrapText(formatChangesetBody(m.cs), m.viewport.Width))
	}
	return m
}

func (m changesetViewModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading changeset..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back")
	}
	if m.cs == nil {
		return "no changeset\n" + footerStyle.Render("esc back")
	}
	closedAt := "(open)"
	if !m.cs.ClosedAt.IsZero() {
		closedAt = m.cs.ClosedAt.Format(time.RFC3339)
	}
	title := m.cs.Comment()
	if title == "" {
		title = fmt.Sprintf("changeset %d", m.cs.ID)
	}
	header := headerStyle.Render(title) + "\n" +
		mutedStyle.Render(fmt.Sprintf("id %d • %s • open %v • created %s • closed %s",
			m.cs.ID, m.cs.User, m.cs.Open, m.cs.CreatedAt.Format(time.RFC3339), closedAt))
	footer := "esc back, x show osmChange xml"
	if m.showXML {
		footer = "esc back, x back to summary"
		if m.xmlLoading {
			footer = "loading xml... esc back"
		}
	}
	return header + "\n\n" + m.viewport.View() + "\n" + footerStyle.Render(footer)
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
