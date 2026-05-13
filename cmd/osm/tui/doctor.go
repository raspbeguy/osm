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
	caps     *api.Capabilities
	capsErr  error
	perms    []string
	permsErr error
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
		caps, capsErr := c.Capabilities(programCtx)
		perms, permsErr := c.Permissions(programCtx)
		return doctorLoadedMsg{caps: caps, capsErr: capsErr, perms: perms, permsErr: permsErr}
	}
}

func (m doctorModel) Update(msg tea.Msg) (doctorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case doctorLoadedMsg:
		m.loading = false
		if msg.capsErr != nil && msg.permsErr != nil {
			m.err = fmt.Errorf("capabilities: %v; permissions: %v", msg.capsErr, msg.permsErr)
		} else {
			m.err = nil
			m.viewport.SetContent(formatDoctor(msg.caps, msg.capsErr, msg.perms, msg.permsErr))
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

func formatDoctor(caps *api.Capabilities, capsErr error, perms []string, permsErr error) string {
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Server") + "\n")
	if capsErr != nil {
		sb.WriteString("  " + errorStyle.Render("error: "+capsErr.Error()) + "\n")
	} else {
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
	}
	sb.WriteString("\n" + headerStyle.Render("Token permissions") + "\n")
	if permsErr != nil {
		sb.WriteString("  " + errorStyle.Render("error: "+permsErr.Error()) + "\n")
	} else {
		for _, p := range perms {
			fmt.Fprintf(&sb, "  %s\n", p)
		}
	}
	return sb.String()
}
