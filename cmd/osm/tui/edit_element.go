package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paulmach/osm"
)

type editElementState int

const (
	editList editElementState = iota
	editAddingTag
	editEditingTag
	editAddingMember
	editEditingMemberRole
)

const (
	focusTags    = 0
	focusMembers = 1
)

type tagItem struct {
	t osm.Tag
}

func (i tagItem) Title() string       { return styledTag(i.t.Key, i.t.Value) }
func (i tagItem) Description() string { return "" }
func (i tagItem) FilterValue() string { return i.t.Key }

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

type editElementModel struct {
	target      *stagedElement
	tagsList    list.Model
	membersList list.Model
	input       textinput.Model
	state       editElementState
	focus       int
	width       int
	height      int
	err         error
}

func newEditElement() editElementModel {
	tl := list.New(nil, newCompactDelegate(), 60, 20)
	tl.Title = "Tags"
	tl.SetShowHelp(false)
	tl.SetShowStatusBar(false)
	tl.SetFilteringEnabled(false)
	ml := list.New(nil, newCompactDelegate(), 60, 20)
	ml.Title = "Members"
	ml.SetShowHelp(false)
	ml.SetShowStatusBar(false)
	ml.SetFilteringEnabled(false)
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 60
	return editElementModel{tagsList: tl, membersList: ml, input: ti}
}

func (m editElementModel) Init() tea.Cmd { return nil }

func (m editElementModel) show(e *stagedElement) editElementModel {
	m.target = e
	m.state = editList
	m.focus = focusTags
	m.err = nil
	m.refreshLists()
	m.resizeLists()
	return m
}

func (m *editElementModel) setSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w - 4
	m.resizeLists()
}

func (m *editElementModel) resizeLists() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	if m.isRelation() {
		leftW, rightW := splitWidths(m.width)
		m.tagsList.SetSize(leftW, m.height)
		m.membersList.SetSize(rightW, m.height)
		return
	}
	m.tagsList.SetSize(m.width, m.height)
	m.membersList.SetSize(0, m.height)
}

func (m *editElementModel) refreshLists() {
	if m.target == nil {
		m.tagsList.SetItems(nil)
		m.membersList.SetItems(nil)
		return
	}
	tagItems := make([]list.Item, len(m.target.Tags))
	for i, t := range m.target.Tags {
		tagItems[i] = tagItem{t: t}
	}
	m.tagsList.SetItems(tagItems)
	memberItems := make([]list.Item, len(m.target.Members))
	for i, mm := range m.target.Members {
		memberItems[i] = memberItem{m: mm}
	}
	m.membersList.SetItems(memberItems)
}

func (m editElementModel) isRelation() bool {
	return m.target != nil && m.target.Kind == "relation"
}

func (m editElementModel) Update(msg tea.Msg) (editElementModel, tea.Cmd) {
	if m.target == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state != editList {
			switch msg.String() {
			case "enter":
				m = m.commitInput()
				return m, nil
			case "esc":
				m.state = editList
				m.err = nil
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "tab":
			if m.isRelation() {
				m.focus = 1 - m.focus
			}
			return m, nil
		case "a":
			if m.focus == focusMembers {
				m.state = editAddingMember
				m.input.Placeholder = "kind id [role]"
			} else {
				m.state = editAddingTag
				m.input.Placeholder = "key=value"
			}
			m.input.SetValue("")
			m.input.Focus()
			m.err = nil
			return m, textinput.Blink
		case "e":
			if m.focus == focusMembers {
				idx := m.membersList.Index()
				if idx >= 0 && idx < len(m.target.Members) {
					m.state = editEditingMemberRole
					m.input.Placeholder = "new role"
					m.input.SetValue(m.target.Members[idx].Role)
					m.input.Focus()
					m.err = nil
					return m, textinput.Blink
				}
				return m, nil
			}
			if t := m.selectedTag(); t != nil {
				m.state = editEditingTag
				m.input.Placeholder = "new value for " + t.Key
				m.input.SetValue(t.Value)
				m.input.Focus()
				m.err = nil
				return m, textinput.Blink
			}
		case "d":
			if m.focus == focusMembers {
				idx := m.membersList.Index()
				if idx >= 0 && idx < len(m.target.Members) {
					m.target.Members = append(m.target.Members[:idx], m.target.Members[idx+1:]...)
					m.refreshLists()
				}
				return m, nil
			}
			idx := m.tagsList.Index()
			if idx >= 0 && idx < len(m.target.Tags) {
				m.target.Tags = append(m.target.Tags[:idx], m.target.Tags[idx+1:]...)
				m.refreshLists()
			}
			return m, nil
		case "K":
			if m.focus != focusMembers {
				break
			}
			idx := m.membersList.Index()
			if idx > 0 && idx < len(m.target.Members) {
				m.target.Members[idx-1], m.target.Members[idx] = m.target.Members[idx], m.target.Members[idx-1]
				m.refreshLists()
				m.membersList.Select(idx - 1)
			}
			return m, nil
		case "J":
			if m.focus != focusMembers {
				break
			}
			idx := m.membersList.Index()
			if idx >= 0 && idx < len(m.target.Members)-1 {
				m.target.Members[idx+1], m.target.Members[idx] = m.target.Members[idx], m.target.Members[idx+1]
				m.refreshLists()
				m.membersList.Select(idx + 1)
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	if m.focus == focusMembers {
		m.membersList, cmd = m.membersList.Update(msg)
	} else {
		m.tagsList, cmd = m.tagsList.Update(msg)
	}
	return m, cmd
}

func (m editElementModel) selectedTag() *osm.Tag {
	idx := m.tagsList.Index()
	if idx < 0 || idx >= len(m.target.Tags) {
		return nil
	}
	return &m.target.Tags[idx]
}

func (m editElementModel) commitInput() editElementModel {
	v := strings.TrimSpace(m.input.Value())
	switch m.state {
	case editAddingTag:
		i := strings.IndexByte(v, '=')
		if i < 0 {
			m.err = errors.New("expected key=value")
			return m
		}
		key := strings.TrimSpace(v[:i])
		val := strings.TrimSpace(v[i+1:])
		if key == "" {
			m.err = errors.New("key required")
			return m
		}
		for j, t := range m.target.Tags {
			if t.Key == key {
				if val == "" {
					m.target.Tags = append(m.target.Tags[:j], m.target.Tags[j+1:]...)
				} else {
					m.target.Tags[j].Value = val
				}
				m.refreshLists()
				m.state = editList
				m.err = nil
				return m
			}
		}
		if val == "" {
			m.err = errors.New("empty value but key does not exist")
			return m
		}
		m.target.Tags = append(m.target.Tags, osm.Tag{Key: key, Value: val})
	case editEditingTag:
		idx := m.tagsList.Index()
		if idx >= 0 && idx < len(m.target.Tags) {
			if v == "" {
				m.target.Tags = append(m.target.Tags[:idx], m.target.Tags[idx+1:]...)
			} else {
				m.target.Tags[idx].Value = v
			}
		}
	case editAddingMember:
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
	case editEditingMemberRole:
		idx := m.membersList.Index()
		if idx >= 0 && idx < len(m.target.Members) {
			m.target.Members[idx].Role = v
		}
	}
	m.refreshLists()
	m.state = editList
	m.err = nil
	return m
}

func (m editElementModel) View() string {
	if m.target == nil {
		return "no element\n" + footerStyle.Render("esc back")
	}
	action := "modify"
	if m.target.Action == stagedCreate {
		action = "create"
	}
	title := fmt.Sprintf("%s %s %d", action, kindGlyph(m.target.Kind), m.target.ID)
	header := headerStyle.Render(title)
	if m.target.Version > 0 {
		header += " " + mutedStyle.Render(fmt.Sprintf("(v%d)", m.target.Version))
	}

	var body, footer string
	switch m.state {
	case editAddingTag:
		body = mutedStyle.Render("Add tag (format: key=value)") + "\n" + m.input.View()
		footer = "enter add, esc cancel"
	case editEditingTag:
		body = mutedStyle.Render("Edit value (empty deletes)") + "\n" + m.input.View()
		footer = "enter save, esc cancel"
	case editAddingMember:
		body = mutedStyle.Render("Add member (kind id [role], e.g. 'way 1234 outer')") + "\n" + m.input.View()
		footer = "enter add, esc cancel"
	case editEditingMemberRole:
		body = mutedStyle.Render("Edit role") + "\n" + m.input.View()
		footer = "enter save, esc cancel"
	default:
		body = m.renderLists()
		footer = m.listFooter()
	}
	if m.err != nil {
		footer = errorStyle.Render(m.err.Error()) + " - " + footer
	}
	return header + "\n\n" + body + "\n" + footerStyle.Render(footer)
}

func (m editElementModel) renderLists() string {
	if !m.isRelation() {
		return m.tagsList.View()
	}
	leftStyle, rightStyle := paneFocused, paneUnfocused
	if m.focus == focusMembers {
		leftStyle, rightStyle = paneUnfocused, paneFocused
	}
	left := leftStyle.Render(m.tagsList.View())
	right := rightStyle.Render(m.membersList.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m editElementModel) listFooter() string {
	if !m.isRelation() {
		return "esc back, a add, e edit, d delete"
	}
	if m.focus == focusMembers {
		return "esc back, tab swap pane, a add, e edit role, d delete, K/J reorder"
	}
	return "esc back, tab swap pane, a add, e edit, d delete"
}
