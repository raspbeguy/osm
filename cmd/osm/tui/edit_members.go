package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"
)

type editMemberState int

const (
	editMemberList editMemberState = iota
	editMemberAdding
	editMemberEditingRole
)

type memberItem struct{ m osm.Member }

func (i memberItem) Title() string {
	role := i.m.Role
	if role == "" {
		role = "(no role)"
	}
	return fmt.Sprintf("%s %d  [%s]", kindGlyph(string(i.m.Type)), i.m.Ref, role)
}

func (i memberItem) Description() string { return "" }
func (i memberItem) FilterValue() string { return i.Title() }

type editMembersModel struct {
	target *stagedElement
	list   list.Model
	input  textinput.Model
	state  editMemberState
	err    error
}

func newEditMembers() editMembersModel {
	l := list.New(nil, newCompactDelegate(), 60, 20)
	l.Title = "Members"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 60
	return editMembersModel{list: l, input: ti}
}

func (m editMembersModel) Init() tea.Cmd { return nil }

func (m editMembersModel) show(e *stagedElement) editMembersModel {
	m.target = e
	m.state = editMemberList
	m.err = nil
	m.refreshList()
	return m
}

func (m *editMembersModel) refreshList() {
	if m.target == nil {
		m.list.SetItems(nil)
		return
	}
	items := make([]list.Item, len(m.target.Members))
	for i, mm := range m.target.Members {
		items[i] = memberItem{m: mm}
	}
	m.list.SetItems(items)
}

func (m editMembersModel) Update(msg tea.Msg) (editMembersModel, tea.Cmd) {
	if m.target == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state != editMemberList {
			switch msg.String() {
			case "enter":
				m = m.commitInput()
				return m, nil
			case "esc":
				m.state = editMemberList
				m.err = nil
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "a":
			m.state = editMemberAdding
			m.input.Placeholder = "kind id [role]"
			m.input.SetValue("")
			m.input.Focus()
			m.err = nil
			return m, textinput.Blink
		case "e":
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.target.Members) {
				m.state = editMemberEditingRole
				m.input.Placeholder = "new role"
				m.input.SetValue(m.target.Members[idx].Role)
				m.input.Focus()
				m.err = nil
				return m, textinput.Blink
			}
		case "d":
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.target.Members) {
				m.target.Members = append(m.target.Members[:idx], m.target.Members[idx+1:]...)
				m.refreshList()
			}
			return m, nil
		case "K":
			idx := m.list.Index()
			if idx > 0 && idx < len(m.target.Members) {
				m.target.Members[idx-1], m.target.Members[idx] = m.target.Members[idx], m.target.Members[idx-1]
				m.refreshList()
				m.list.Select(idx - 1)
			}
			return m, nil
		case "J":
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.target.Members)-1 {
				m.target.Members[idx+1], m.target.Members[idx] = m.target.Members[idx], m.target.Members[idx+1]
				m.refreshList()
				m.list.Select(idx + 1)
			}
			return m, nil
		case "t":
			return m, func() tea.Msg {
				return navigateMsg{to: screenEditElement, parent: screenComposeChangeset}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m editMembersModel) commitInput() editMembersModel {
	v := strings.TrimSpace(m.input.Value())
	switch m.state {
	case editMemberAdding:
		parts := strings.Fields(v)
		if len(parts) < 2 {
			m.err = errors.New("expected: kind id [role]")
			return m
		}
		kind := parts[0]
		if kind != "node" && kind != "way" && kind != "relation" {
			m.err = errors.New("kind must be node, way, or relation")
			return m
		}
		id, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			m.err = fmt.Errorf("invalid id %q", parts[1])
			return m
		}
		role := ""
		if len(parts) > 2 {
			role = strings.Join(parts[2:], " ")
		}
		m.target.Members = append(m.target.Members, osm.Member{
			Type: osm.Type(kind),
			Ref:  id,
			Role: role,
		})
		m.refreshList()
	case editMemberEditingRole:
		idx := m.list.Index()
		if idx >= 0 && idx < len(m.target.Members) {
			m.target.Members[idx].Role = v
			m.refreshList()
		}
	}
	m.state = editMemberList
	m.err = nil
	return m
}

func (m editMembersModel) View() string {
	if m.target == nil {
		return "no element\n" + footerStyle.Render("esc back")
	}
	title := fmt.Sprintf("members of %s %d", kindGlyph(m.target.Kind), m.target.ID)
	header := headerStyle.Render(title)
	var body, footer string
	switch m.state {
	case editMemberAdding:
		body = mutedStyle.Render("Add member (kind id [role], e.g. 'way 1234 outer')") + "\n" + m.input.View()
		footer = "enter add, esc cancel"
	case editMemberEditingRole:
		body = mutedStyle.Render("Edit role") + "\n" + m.input.View()
		footer = "enter save, esc cancel"
	default:
		body = m.list.View()
		footer = "esc back, a add, e edit role, d delete, K/J reorder, t tags"
	}
	if m.err != nil {
		footer = errorStyle.Render(m.err.Error()) + " — " + footer
	}
	return header + "\n\n" + body + "\n" + footerStyle.Render(footer)
}
