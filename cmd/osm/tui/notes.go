package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type notesState int

const (
	notesStateInput notesState = iota
	notesStateLoading
	notesStateNote
	notesStateList
	notesStateError
)

type noteLoadedMsg struct {
	note  *api.Note
	notes []*api.Note
	err   error
}

type noteItem struct{ n *api.Note }

func (i noteItem) Title() string {
	return fmt.Sprintf("%d  [%s]", i.n.ID, i.n.Status)
}

func (i noteItem) Description() string {
	if len(i.n.Comments) > 0 {
		return i.n.Comments[0].Text
	}
	return "(no comment)"
}

func (i noteItem) FilterValue() string { return strconv.FormatInt(i.n.ID, 10) }

type notesModel struct {
	client   *api.Client
	state    notesState
	input    textinput.Model
	spinner  spinner.Model
	viewport viewport.Model
	list     list.Model
	err      error
	note     *api.Note
}

func (m notesModel) rewrap() notesModel {
	if m.note != nil && m.state == notesStateNote {
		m.viewport.SetContent(wrapText(formatNote(m.note), m.viewport.Width))
	}
	return m
}

func newNotes(c *api.Client) notesModel {
	ti := textinput.New()
	ti.Placeholder = "note ID or bbox l,b,r,t"
	ti.CharLimit = 200
	ti.Width = 60
	s := spinner.New()
	s.Spinner = spinner.Dot
	l := list.New(nil, list.NewDefaultDelegate(), 60, 20)
	l.Title = "Notes"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return notesModel{
		client:   c,
		state:    notesStateInput,
		input:    ti,
		spinner:  s,
		viewport: viewport.New(60, 15),
		list:     l,
	}
}

func (m notesModel) Init() tea.Cmd { return nil }

func (m notesModel) show() (notesModel, tea.Cmd) {
	m.state = notesStateInput
	m.err = nil
	m.input.SetValue("")
	m.input.Focus()
	return m, textinput.Blink
}

func (m notesModel) query(input string) tea.Cmd {
	id, bbox, isID, err := parseNoteQuery(input)
	if err != nil {
		e := err
		return func() tea.Msg { return noteLoadedMsg{err: e} }
	}
	client := m.client
	if isID {
		return func() tea.Msg {
			n, err := client.GetNote(context.Background(), id)
			return noteLoadedMsg{note: n, err: err}
		}
	}
	return func() tea.Msg {
		ns, err := client.QueryNotes(context.Background(), api.NotesQuery{BBox: bbox})
		return noteLoadedMsg{notes: ns, err: err}
	}
}

func (m notesModel) Update(msg tea.Msg) (notesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case noteLoadedMsg:
		if msg.err != nil {
			m.state = notesStateError
			m.err = msg.err
			return m, nil
		}
		if msg.note != nil {
			m.state = notesStateNote
			m.note = msg.note
			m.viewport.SetContent(wrapText(formatNote(m.note), m.viewport.Width))
			return m, nil
		}
		m.state = notesStateList
		items := make([]list.Item, len(msg.notes))
		for i, n := range msg.notes {
			items[i] = noteItem{n: n}
		}
		m.list.SetItems(items)
		return m, nil
	case spinner.TickMsg:
		if m.state != notesStateLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch m.state {
		case notesStateInput:
			if msg.Type == tea.KeyEnter {
				v := strings.TrimSpace(m.input.Value())
				if v == "" {
					return m, nil
				}
				m.state = notesStateLoading
				return m, tea.Batch(m.spinner.Tick, m.query(v))
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		case notesStateError, notesStateNote, notesStateList:
			if k := msg.String(); k == "/" || k == "i" {
				return m.show()
			}
		}
	}
	switch m.state {
	case notesStateInput:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case notesStateNote:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	case notesStateList:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m notesModel) View() string {
	switch m.state {
	case notesStateInput:
		return headerStyle.Render("Notes lookup") + "\n" +
			mutedStyle.Render("enter a note ID (e.g. 329125) or a bbox (l,b,r,t)") + "\n\n" +
			m.input.View() + "\n" +
			footerStyle.Render("enter submit, esc back")
	case notesStateLoading:
		return m.spinner.View() + " loading..."
	case notesStateError:
		return errorStyle.Render("error: "+m.err.Error()) + "\n" + footerStyle.Render("/ new query, esc back")
	case notesStateNote:
		return m.viewport.View() + "\n" + footerStyle.Render("/ new query, esc back")
	case notesStateList:
		return m.list.View() + "\n" + footerStyle.Render("/ new query, esc back")
	}
	return ""
}

func formatNote(n *api.Note) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "ID: %d\nStatus: %s\nLocation: %g, %g\nCreated: %s\n", n.ID, n.Status, n.Lat, n.Lon, n.CreatedAt)
	if n.ClosedAt != "" {
		fmt.Fprintf(&sb, "Closed: %s\n", n.ClosedAt)
	}
	sb.WriteString("\nComments:\n")
	for _, c := range n.Comments {
		fmt.Fprintf(&sb, "\n[%s] %s by %s:\n%s\n", c.Date, c.Action, c.User, c.Text)
	}
	return sb.String()
}

// parseNoteQuery decides whether s is a numeric note ID or a four-float bbox.
func parseNoteQuery(s string) (id int64, bbox [4]float64, isID bool, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, [4]float64{}, false, fmt.Errorf("empty input")
	}
	if !strings.Contains(s, ",") {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, [4]float64{}, false, fmt.Errorf("not a note ID or bbox: %q", s)
		}
		return n, [4]float64{}, true, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, [4]float64{}, false, fmt.Errorf("bbox needs 4 comma-separated values: l,b,r,t")
	}
	var bb [4]float64
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return 0, [4]float64{}, false, fmt.Errorf("bbox value %d (%q): %w", i+1, p, err)
		}
		bb[i] = v
	}
	return 0, bb, false, nil
}
