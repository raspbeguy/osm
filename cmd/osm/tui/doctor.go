package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type doctorLoadedMsg struct {
	caps  *api.Capabilities
	perms []string
	err   error
}

type doctorModel struct {
	client   *api.Client
	spinner  spinner.Model
	viewport viewport.Model
	err      error
	loading  bool
}

func newDoctor(c *api.Client) doctorModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return doctorModel{
		client:   c,
		spinner:  s,
		viewport: viewport.New(60, 20),
	}
}

func (m doctorModel) Init() tea.Cmd { return nil }

func (m doctorModel) show() (doctorModel, tea.Cmd) {
	m.loading = true
	m.err = nil
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m doctorModel) load() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		caps, err := c.Capabilities(programCtx)
		if err != nil {
			return doctorLoadedMsg{err: err}
		}
		perms, err := c.Permissions(programCtx)
		if err != nil {
			return doctorLoadedMsg{err: err}
		}
		return doctorLoadedMsg{caps: caps, perms: perms}
	}
}

func (m doctorModel) Update(msg tea.Msg) (doctorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case doctorLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.viewport.SetContent(formatDoctor(msg.caps, msg.perms))
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

func (m doctorModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back")
	}
	return m.viewport.View() + "\n" + footerStyle.Render("esc back")
}

func formatDoctor(caps *api.Capabilities, perms []string) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Server") + "\n")
	fmt.Fprintf(&sb, "  api version       %s..%s\n", caps.MinAPIVersion, caps.MaxAPIVersion)
	fmt.Fprintf(&sb, "  status            db=%s api=%s gpx=%s\n", caps.DatabaseStatus, caps.APIStatus, caps.GPXStatus)
	fmt.Fprintf(&sb, "  area max          %g deg^2\n", caps.AreaMax)
	fmt.Fprintf(&sb, "  note area max     %g\n", caps.NoteAreaMax)
	fmt.Fprintf(&sb, "  changeset max     %d elements, query limit %d\n", caps.ChangesetMaxElements, caps.ChangesetMaxQuery)
	fmt.Fprintf(&sb, "  way nodes max     %d\n", caps.WayNodesMax)
	fmt.Fprintf(&sb, "  relation members  %d max\n", caps.RelationMembersMax)
	fmt.Fprintf(&sb, "  notes query max   %d\n", caps.NotesMaxQuery)
	fmt.Fprintf(&sb, "  tracepoints page  %d\n", caps.TracepointsPerPage)
	fmt.Fprintf(&sb, "  timeout           %d s\n", caps.TimeoutSeconds)
	sb.WriteString("\n" + headerStyle.Render("Token permissions") + "\n")
	for _, p := range perms {
		fmt.Fprintf(&sb, "  %s\n", p)
	}
	return sb.String()
}
