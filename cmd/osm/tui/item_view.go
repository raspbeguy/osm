package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type itemViewModel struct {
	elem changesetElement
}

func newItemView() itemViewModel { return itemViewModel{} }

func (m itemViewModel) Init() tea.Cmd { return nil }

func (m itemViewModel) show(e changesetElement) itemViewModel {
	m.elem = e
	return m
}

func (m itemViewModel) Update(msg tea.Msg) (itemViewModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "h" && m.elem.ID != 0 {
		e := m.elem
		return m, func() tea.Msg {
			return navigateMsg{to: screenHistory, itemID: e.ID, kind: e.Kind, parent: screenItemView}
		}
	}
	return m, nil
}

func (m itemViewModel) View() string {
	if m.elem.ID == 0 {
		return "no element\n" + footerStyle.Render("esc back")
	}
	var actionWord string
	switch m.elem.Action {
	case '+':
		actionWord = "created"
	case '-':
		actionWord = "deleted"
	case '~':
		actionWord = "modified"
	}
	var sb strings.Builder
	title := fmt.Sprintf("%c %s %d", m.elem.Action, m.elem.Kind, m.elem.ID)
	sb.WriteString(headerStyle.Render(title) + "\n")
	sb.WriteString(mutedStyle.Render(fmt.Sprintf("version %d • %s", m.elem.Version, actionWord)) + "\n\n")
	if len(m.elem.Tags) == 0 {
		sb.WriteString("(no tags)\n")
	} else {
		sb.WriteString(headerStyle.Render("Tags") + "\n")
		for _, t := range m.elem.Tags {
			fmt.Fprintf(&sb, "  %s = %s\n", t.Key, t.Value)
		}
	}
	sb.WriteString("\n" + footerStyle.Render("esc back, h view history"))
	return sb.String()
}
