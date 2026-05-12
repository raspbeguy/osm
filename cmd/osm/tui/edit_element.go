package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// editElementModel is phase 1: a read-only view of a staged element's current
// tags (and members if it's a relation). Editing widgets land in later phases.
type editElementModel struct {
	target   *stagedElement
	viewport viewport.Model
}

func newEditElement() editElementModel {
	return editElementModel{viewport: viewport.New(60, 15)}
}

func (m editElementModel) Init() tea.Cmd { return nil }

func (m editElementModel) show(e *stagedElement) editElementModel {
	m.target = e
	return m.rewrap()
}

func (m editElementModel) rewrap() editElementModel {
	if m.target == nil {
		m.viewport.SetContent(mutedStyle.Render("(no element)"))
		return m
	}
	m.viewport.SetContent(wrapText(renderStagedElement(m.target), m.viewport.Width))
	return m
}

func renderStagedElement(e *stagedElement) string {
	var sb strings.Builder
	action := "modify"
	if e.Action == stagedCreate {
		action = "create"
	}
	sb.WriteString(headerStyle.Render(fmt.Sprintf("%s %s %d", action, e.Kind, e.ID)) + "\n")
	if e.Version > 0 {
		sb.WriteString(mutedStyle.Render(fmt.Sprintf("version %d", e.Version)) + "\n")
	}
	sb.WriteString("\n" + headerStyle.Render("Tags") + "\n")
	if len(e.Tags) == 0 {
		sb.WriteString(mutedStyle.Render("  (none)") + "\n")
	} else {
		for _, t := range e.Tags {
			fmt.Fprintf(&sb, "  %s = %s\n", t.Key, t.Value)
		}
	}
	if e.Kind == "relation" {
		sb.WriteString("\n" + headerStyle.Render("Members") + "\n")
		if len(e.Members) == 0 {
			sb.WriteString(mutedStyle.Render("  (none)") + "\n")
		} else {
			for _, mm := range e.Members {
				role := mm.Role
				if role == "" {
					role = "(no role)"
				}
				fmt.Fprintf(&sb, "  %s %d  [%s]\n", mm.Type, mm.Ref, role)
			}
		}
	}
	return sb.String()
}

func (m editElementModel) Update(msg tea.Msg) (editElementModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m editElementModel) View() string {
	if m.target == nil {
		return "no element\n" + footerStyle.Render("esc back")
	}
	footer := "esc back (editing arrives in next phase)"
	return m.viewport.View() + "\n" + footerStyle.Render(footer)
}
