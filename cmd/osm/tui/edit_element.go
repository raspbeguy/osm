package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/paulmach/osm"
)

type editTagState int

const (
	editTagList editTagState = iota
	editTagAdding
	editTagEditing
)

type tagItem struct {
	t osm.Tag
}

func (i tagItem) Title() string       { return styledTag(i.t.Key, i.t.Value) }
func (i tagItem) Description() string { return "" }
func (i tagItem) FilterValue() string { return i.t.Key }

type editElementModel struct {
	target *stagedElement
	list   list.Model
	input  textinput.Model
	state  editTagState
	err    error
}

func newEditElement() editElementModel {
	l := list.New(nil, newCompactDelegate(), 60, 20)
	l.Title = "Tags"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 60
	return editElementModel{list: l, input: ti}
}

func (m editElementModel) Init() tea.Cmd { return nil }

func (m editElementModel) show(e *stagedElement) editElementModel {
	m.target = e
	m.state = editTagList
	m.err = nil
	m.refreshList()
	return m
}

func (m *editElementModel) refreshList() {
	if m.target == nil {
		m.list.SetItems(nil)
		return
	}
	items := make([]list.Item, len(m.target.Tags))
	for i, t := range m.target.Tags {
		items[i] = tagItem{t: t}
	}
	m.list.SetItems(items)
}

func (m editElementModel) Update(msg tea.Msg) (editElementModel, tea.Cmd) {
	if m.target == nil {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state != editTagList {
			switch msg.String() {
			case "enter":
				m = m.commitInput()
				return m, nil
			case "esc":
				m.state = editTagList
				m.err = nil
				return m, nil
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		switch msg.String() {
		case "a":
			m.state = editTagAdding
			m.input.Placeholder = "key=value"
			m.input.SetValue("")
			m.input.Focus()
			m.err = nil
			return m, textinput.Blink
		case "e":
			if t := m.selectedTag(); t != nil {
				m.state = editTagEditing
				m.input.Placeholder = "new value for " + t.Key
				m.input.SetValue(t.Value)
				m.input.Focus()
				m.err = nil
				return m, textinput.Blink
			}
		case "d":
			idx := m.list.Index()
			if idx >= 0 && idx < len(m.target.Tags) {
				m.target.Tags = append(m.target.Tags[:idx], m.target.Tags[idx+1:]...)
				m.refreshList()
			}
			return m, nil
		case "m":
			if m.target.Kind == "relation" {
				return m, func() tea.Msg {
					return navigateMsg{to: screenEditMembers}
				}
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m editElementModel) selectedTag() *osm.Tag {
	idx := m.list.Index()
	if idx < 0 || idx >= len(m.target.Tags) {
		return nil
	}
	return &m.target.Tags[idx]
}

func (m editElementModel) commitInput() editElementModel {
	v := strings.TrimSpace(m.input.Value())
	switch m.state {
	case editTagAdding:
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
				m.refreshList()
				m.state = editTagList
				m.err = nil
				return m
			}
		}
		if val == "" {
			m.err = errors.New("empty value but key does not exist")
			return m
		}
		m.target.Tags = append(m.target.Tags, osm.Tag{Key: key, Value: val})
	case editTagEditing:
		idx := m.list.Index()
		if idx >= 0 && idx < len(m.target.Tags) {
			if v == "" {
				m.target.Tags = append(m.target.Tags[:idx], m.target.Tags[idx+1:]...)
			} else {
				m.target.Tags[idx].Value = v
			}
		}
	}
	m.refreshList()
	m.state = editTagList
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
	case editTagAdding:
		body = mutedStyle.Render("Add tag (format: key=value)") + "\n" + m.input.View()
		footer = "enter add, esc cancel"
	case editTagEditing:
		body = mutedStyle.Render("Edit value") + "\n" + m.input.View()
		footer = "enter save, esc cancel"
	default:
		body = m.list.View()
		footer = "esc back, a add, e edit, d delete"
		if m.target.Kind == "relation" {
			footer += ", m members"
		}
	}
	if m.err != nil {
		footer = errorStyle.Render(m.err.Error()) + " - " + footer
	}
	return header + "\n\n" + body + "\n" + footerStyle.Render(footer)
}
