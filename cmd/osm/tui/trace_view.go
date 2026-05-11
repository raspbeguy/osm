package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type traceViewLoadedMsg struct {
	traceID int64
	trace   *api.Trace
	err     error
}

type traceDataLoadedMsg struct {
	traceID int64
	data    string
	err     error
}

type traceViewModel struct {
	client      *api.Client
	traceID     int64
	spinner     spinner.Model
	viewport    viewport.Model
	trace       *api.Trace
	err         error
	loading     bool
	data        string
	dataLoading bool
	showData    bool
}

func newTraceView(c *api.Client) traceViewModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return traceViewModel{
		client:   c,
		spinner:  s,
		viewport: viewport.New(60, 15),
	}
}

func (m traceViewModel) Init() tea.Cmd { return nil }

func (m traceViewModel) show(id int64) (traceViewModel, tea.Cmd) {
	m.traceID = id
	m.loading = true
	m.err = nil
	m.trace = nil
	m.data = ""
	m.dataLoading = false
	m.showData = false
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m traceViewModel) load() tea.Cmd {
	client := m.client
	id := m.traceID
	return func() tea.Msg {
		t, err := client.GetTrace(context.Background(), id)
		return traceViewLoadedMsg{traceID: id, trace: t, err: err}
	}
}

func (m traceViewModel) loadData() tea.Cmd {
	client := m.client
	id := m.traceID
	return func() tea.Msg {
		d, err := client.GetTraceData(context.Background(), id)
		return traceDataLoadedMsg{traceID: id, data: d, err: err}
	}
}

func (m traceViewModel) rewrap() traceViewModel {
	if m.showData && m.data != "" {
		m.viewport.SetContent(wrapText(m.data, m.viewport.Width))
	} else if m.trace != nil {
		m.viewport.SetContent(wrapText(formatTraceBody(m.trace), m.viewport.Width))
	}
	return m
}

func (m traceViewModel) Update(msg tea.Msg) (traceViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case traceViewLoadedMsg:
		if msg.traceID != m.traceID {
			return m, nil
		}
		m.loading = false
		m.trace = msg.trace
		m.err = msg.err
		m = m.rewrap()
		return m, nil
	case traceDataLoadedMsg:
		if msg.traceID != m.traceID {
			return m, nil
		}
		m.dataLoading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.data = msg.data
		if m.showData {
			m.viewport.SetContent(wrapText(m.data, m.viewport.Width))
		}
		return m, nil
	case spinner.TickMsg:
		if !m.loading && !m.dataLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "d" && m.trace != nil {
			m.showData = !m.showData
			m = m.rewrap()
			if m.showData && m.data == "" && !m.dataLoading {
				m.dataLoading = true
				return m, tea.Batch(m.spinner.Tick, m.loadData())
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m traceViewModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading trace..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back")
	}
	if m.trace == nil {
		return "no trace\n" + footerStyle.Render("esc back")
	}
	title := m.trace.Description
	if title == "" {
		title = m.trace.Name
	}
	header := headerStyle.Render(title) + "\n" +
		mutedStyle.Render(fmt.Sprintf("id %d • %s • %s • %s", m.trace.ID, m.trace.User, m.trace.Visibility, m.trace.Timestamp))
	footer := "esc back, d show raw gpx"
	if m.showData {
		footer = "esc back, d back to summary"
		if m.dataLoading {
			footer = "loading gpx... esc back"
		}
	}
	return header + "\n\n" + m.viewport.View() + "\n" + footerStyle.Render(footer)
}

func formatTraceBody(t *api.Trace) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Name: %s\n", t.Name)
	fmt.Fprintf(&sb, "Location: %g, %g\n", t.Lat, t.Lon)
	fmt.Fprintf(&sb, "Pending: %v\n", t.Pending)
	if len(t.Tags) > 0 {
		fmt.Fprintf(&sb, "Tags: %s\n", strings.Join(t.Tags, ", "))
	}
	if t.Description != "" {
		fmt.Fprintf(&sb, "\n%s\n", t.Description)
	}
	return sb.String()
}
