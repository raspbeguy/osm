package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// newCompactDelegate returns a list delegate that renders each item on a
// single line: title only, no description, no inter-item spacing.
func newCompactDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	return d
}

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	footerStyle   = lipgloss.NewStyle().Faint(true).MarginTop(1)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	mutedStyle    = lipgloss.NewStyle().Faint(true)
	paneFocused   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("12"))
	paneUnfocused = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8"))
)

// wrapText soft-wraps s to width columns, preserving paragraphs.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}
