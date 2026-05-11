package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
)

type historyState int

const (
	historyStateInput historyState = iota
	historyStateLoading
	historyStateResult
	historyStateError
)

type historyLoadedMsg struct {
	text string
	err  error
}

type historyModel struct {
	client   *api.Client
	state    historyState
	input    textinput.Model
	spinner  spinner.Model
	viewport viewport.Model
	err      error
	rawText  string
}

func newHistory(c *api.Client) historyModel {
	ti := textinput.New()
	ti.Placeholder = "node|way|relation <id>"
	ti.CharLimit = 100
	ti.Width = 40
	s := spinner.New()
	s.Spinner = spinner.Dot
	return historyModel{
		client:   c,
		state:    historyStateInput,
		input:    ti,
		spinner:  s,
		viewport: viewport.New(60, 15),
	}
}

func (m historyModel) Init() tea.Cmd { return nil }

func (m historyModel) show() (historyModel, tea.Cmd) {
	m.state = historyStateInput
	m.err = nil
	m.input.SetValue("")
	m.input.Focus()
	return m, textinput.Blink
}

func (m historyModel) rewrap() historyModel {
	if m.state == historyStateResult && m.rawText != "" {
		m.viewport.SetContent(wrapText(m.rawText, m.viewport.Width))
	}
	return m
}

func (m historyModel) load(input string) tea.Cmd {
	kind, id, err := parseHistoryQuery(input)
	if err != nil {
		e := err
		return func() tea.Msg { return historyLoadedMsg{err: e} }
	}
	client := m.client
	return func() tea.Msg {
		text, err := fetchHistory(client, kind, id)
		return historyLoadedMsg{text: text, err: err}
	}
}

func fetchHistory(c *api.Client, kind string, id int64) (string, error) {
	var lines []string
	switch kind {
	case "node":
		ns, err := c.NodeHistory(context.Background(), osm.NodeID(id))
		if err != nil {
			return "", err
		}
		for _, n := range ns {
			lines = append(lines, formatHistoryRow(int(n.Version), n.Timestamp.Format("2006-01-02T15:04:05Z"), int64(n.ChangesetID), n.Visible, n.User))
		}
	case "way":
		ws, err := c.WayHistory(context.Background(), osm.WayID(id))
		if err != nil {
			return "", err
		}
		for _, w := range ws {
			lines = append(lines, formatHistoryRow(int(w.Version), w.Timestamp.Format("2006-01-02T15:04:05Z"), int64(w.ChangesetID), w.Visible, w.User))
		}
	case "relation":
		rs, err := c.RelationHistory(context.Background(), osm.RelationID(id))
		if err != nil {
			return "", err
		}
		for _, r := range rs {
			lines = append(lines, formatHistoryRow(int(r.Version), r.Timestamp.Format("2006-01-02T15:04:05Z"), int64(r.ChangesetID), r.Visible, r.User))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func formatHistoryRow(version int, ts string, cs int64, visible bool, user string) string {
	state := "visible"
	if !visible {
		state = "deleted"
	}
	return fmt.Sprintf("v%-3d  %s  cs=%d  %s  %s", version, ts, cs, state, user)
}

func parseHistoryQuery(s string) (string, int64, error) {
	parts := strings.Fields(strings.TrimSpace(s))
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("expected 'node|way|relation <id>'")
	}
	if parts[0] != "node" && parts[0] != "way" && parts[0] != "relation" {
		return "", 0, fmt.Errorf("kind must be node, way, or relation; got %q", parts[0])
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("not a numeric id: %q", parts[1])
	}
	return parts[0], id, nil
}

func (m historyModel) Update(msg tea.Msg) (historyModel, tea.Cmd) {
	switch msg := msg.(type) {
	case historyLoadedMsg:
		if msg.err != nil {
			m.state = historyStateError
			m.err = msg.err
			return m, nil
		}
		m.state = historyStateResult
		m.rawText = msg.text
		m.viewport.SetContent(wrapText(m.rawText, m.viewport.Width))
		return m, nil
	case spinner.TickMsg:
		if m.state != historyStateLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch m.state {
		case historyStateInput:
			if msg.Type == tea.KeyEnter {
				v := strings.TrimSpace(m.input.Value())
				if v == "" {
					return m, nil
				}
				m.state = historyStateLoading
				return m, tea.Batch(m.spinner.Tick, m.load(v))
			}
		case historyStateResult, historyStateError:
			if k := msg.String(); k == "/" || k == "i" {
				return m.show()
			}
		}
	}
	switch m.state {
	case historyStateInput:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case historyStateResult:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m historyModel) View() string {
	switch m.state {
	case historyStateInput:
		return headerStyle.Render("Element history") + "\n" +
			mutedStyle.Render("enter 'node 123', 'way 456', or 'relation 789'") + "\n\n" +
			m.input.View() + "\n" +
			footerStyle.Render("enter submit, esc back")
	case historyStateLoading:
		return m.spinner.View() + " loading..."
	case historyStateError:
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("/ new query, esc back")
	case historyStateResult:
		return m.viewport.View() + "\n" + footerStyle.Render("/ new query, esc back")
	}
	return ""
}
