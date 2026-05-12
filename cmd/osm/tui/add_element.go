package tui

import (
	"errors"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

type addElementLoadedMsg struct {
	elem *stagedElement
	err  error
}

type addElementModel struct {
	client  *api.Client
	input   textinput.Model
	spinner spinner.Model
	err     error
	loading bool
}

func newAddElement(c *api.Client) addElementModel {
	ti := textinput.New()
	ti.Placeholder = "node|way|relation <id>"
	ti.CharLimit = 100
	ti.Width = 40
	s := spinner.New()
	s.Spinner = spinner.Dot
	return addElementModel{client: c, input: ti, spinner: s}
}

func (m addElementModel) Init() tea.Cmd { return nil }

func (m addElementModel) show() (addElementModel, tea.Cmd) {
	m.err = nil
	m.loading = false
	m.input.SetValue("")
	m.input.Focus()
	return m, textinput.Blink
}

func (m addElementModel) submit() tea.Cmd {
	v := strings.TrimSpace(m.input.Value())
	if v == "" {
		return nil
	}
	parts := strings.Fields(v)
	if len(parts) != 2 {
		err := errParseAddElement
		return func() tea.Msg { return addElementLoadedMsg{err: err} }
	}
	kind := parts[0]
	if kind != "node" && kind != "way" && kind != "relation" {
		err := errParseAddElement
		return func() tea.Msg { return addElementLoadedMsg{err: err} }
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		err := errParseAddElement
		return func() tea.Msg { return addElementLoadedMsg{err: err} }
	}
	client := m.client
	return func() tea.Msg {
		e, err := fetchStagedFor(client, kind, id)
		return addElementLoadedMsg{elem: e, err: err}
	}
}

var errParseAddElement = parseAddElementError{}

type parseAddElementError struct{}

func (parseAddElementError) Error() string {
	return "expected: node|way|relation <id>"
}

func (m addElementModel) Update(msg tea.Msg) (addElementModel, tea.Cmd) {
	switch msg := msg.(type) {
	case addElementLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.elem == nil {
			m.err = errors.New("element not found")
			return m, nil
		}
		elem := msg.elem
		return m, func() tea.Msg { return stagedAddedMsg{elem: elem} }
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.Type == tea.KeyEnter && !m.loading {
			m.err = nil
			cmd := m.submit()
			if cmd != nil {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, cmd)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m addElementModel) View() string {
	if m.loading {
		return m.spinner.View() + " fetching element..."
	}
	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Add element to changeset") + "\n")
	sb.WriteString(mutedStyle.Render("enter element kind and id (e.g. 'node 11724474473')") + "\n\n")
	if m.err != nil {
		sb.WriteString(errorStyle.Render("✗ "+m.err.Error()) + "\n\n")
	}
	sb.WriteString(m.input.View())
	return sb.String() + "\n" + footerStyle.Render("enter submit, esc cancel")
}
