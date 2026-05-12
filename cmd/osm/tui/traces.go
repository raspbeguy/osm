package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/raspbeguy/osm/api"
)

type traceItem struct{ t *api.Trace }

func (i traceItem) Title() string {
	desc := i.t.Description
	if desc == "" {
		desc = i.t.Name
	}
	date := i.t.Timestamp
	if len(date) >= 10 {
		date = date[:10]
	}
	return fmt.Sprintf("%s  %d  [%s]  %s", date, i.t.ID, i.t.Visibility, desc)
}

func (i traceItem) Description() string { return "" }

func (i traceItem) FilterValue() string { return i.Title() }

type tracesLoadedMsg struct {
	traces []*api.Trace
	err    error
}

type traceDataLoadedMsg struct {
	id   int64
	data string
	err  error
}

type tracesModel struct {
	client      *api.Client
	spinner     spinner.Model
	list        list.Model
	viewport    viewport.Model
	data        map[int64]string
	dataLoading map[int64]bool
	err         error
	loading     bool
	focus       int  // 0=list, 1=detail
	showData    bool // false=summary, true=raw gpx
	lastID      int64
}

func newTraces(c *api.Client) tracesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, newCompactDelegate(), 40, 20)
	l.Title = "Traces"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return tracesModel{
		client:      c,
		spinner:     s,
		list:        l,
		viewport:    viewport.New(40, 20),
		data:        map[int64]string{},
		dataLoading: map[int64]bool{},
	}
}

func (m tracesModel) Init() tea.Cmd { return nil }

func (m tracesModel) show() (tracesModel, tea.Cmd) {
	m.loading = true
	m.err = nil
	return m, tea.Batch(m.spinner.Tick, m.load())
}

func (m tracesModel) load() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		ts, err := client.ListTraces(programCtx)
		return tracesLoadedMsg{traces: ts, err: err}
	}
}

func (m tracesModel) fetchData(id int64) tea.Cmd {
	client := m.client
	return func() tea.Msg {
		d, err := client.GetTraceData(programCtx, id)
		return traceDataLoadedMsg{id: id, data: d, err: err}
	}
}

func (m tracesModel) currentID() int64 {
	if i, ok := m.list.SelectedItem().(traceItem); ok {
		return i.t.ID
	}
	return 0
}

func (m tracesModel) currentTrace() *api.Trace {
	if i, ok := m.list.SelectedItem().(traceItem); ok {
		return i.t
	}
	return nil
}

func (m tracesModel) ensureData(id int64) tea.Cmd {
	if id == 0 || !m.showData {
		return nil
	}
	if _, ok := m.data[id]; ok {
		return nil
	}
	if m.dataLoading[id] {
		return nil
	}
	m.dataLoading[id] = true
	return tea.Batch(m.spinner.Tick, m.fetchData(id))
}

func (m tracesModel) Update(msg tea.Msg) (tracesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tracesLoadedMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			items := make([]list.Item, len(msg.traces))
			for i, t := range msg.traces {
				items[i] = traceItem{t: t}
			}
			m.list.SetItems(items)
		}
		m.lastID = m.currentID()
		cmd := m.ensureData(m.lastID)
		m = m.rewrap()
		return m, cmd
	case traceDataLoadedMsg:
		delete(m.dataLoading, msg.id)
		if msg.err == nil {
			m.data[msg.id] = msg.data
		}
		m = m.rewrap()
		return m, nil
	case spinner.TickMsg:
		if !m.loading && len(m.dataLoading) == 0 {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if !inFilter(m.list) {
			switch msg.String() {
			case "r":
				return m.show()
			case "tab":
				m.focus = 1 - m.focus
				return m, nil
			case "d":
				m.showData = !m.showData
				cmd := m.ensureData(m.currentID())
				m = m.rewrap()
				return m, cmd
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
		fetchCmd = m.ensureData(curID)
		m = m.rewrap()
	}
	return m, tea.Batch(cmd, fetchCmd)
}

func (m tracesModel) rewrap() tracesModel {
	id := m.currentID()
	if id == 0 {
		m.viewport.SetContent(mutedStyle.Render("(no selection)"))
		return m
	}
	t := m.currentTrace()
	if t == nil {
		m.viewport.SetContent(mutedStyle.Render("(no trace)"))
		return m
	}
	if m.showData {
		if m.dataLoading[id] {
			m.viewport.SetContent(m.spinner.View() + " loading gpx...")
			return m
		}
		if data, ok := m.data[id]; ok {
			m.viewport.SetContent(wrapText(highlightXML(data), m.viewport.Width))
			return m
		}
		m.viewport.SetContent(mutedStyle.Render("(no data yet)"))
		return m
	}
	m.viewport.SetContent(wrapText(formatTraceDetail(t), m.viewport.Width))
	return m
}

func formatTraceDetail(t *api.Trace) string {
	var sb strings.Builder
	title := t.Description
	if title == "" {
		title = t.Name
	}
	sb.WriteString(headerStyle.Render(title) + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("id %d • %s • %s • %s", t.ID, t.User, t.Visibility, t.Timestamp)) + "\n\n")
	fmt.Fprintf(&sb, "Name: %s\n", t.Name)
	fmt.Fprintf(&sb, "Pending: %v\n", t.Pending)
	fmt.Fprintf(&sb, "Location: %g, %g\n", t.Lat, t.Lon)
	if len(t.Tags) > 0 {
		fmt.Fprintf(&sb, "Tags: %s\n", strings.Join(t.Tags, ", "))
	}
	return sb.String()
}

func (m tracesModel) View() string {
	if m.loading {
		return m.spinner.View() + " loading traces..."
	}
	if m.err != nil {
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("esc back, r retry")
	}
	if len(m.list.Items()) == 0 {
		return mutedStyle.Render("(no traces)") + "\n" + footerStyle.Render("esc back, r refresh")
	}
	leftStyle, rightStyle := paneFocused, paneUnfocused
	if m.focus == 1 {
		leftStyle, rightStyle = paneUnfocused, paneFocused
	}
	left := leftStyle.Render(m.list.View())
	right := rightStyle.Render(m.viewport.View())
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	mode := "summary"
	if m.showData {
		mode = "raw gpx"
	}
	footer := fmt.Sprintf("esc back, tab swap pane, d toggle (%s), r refresh", mode)
	return body + "\n" + footerStyle.Render(footer)
}
