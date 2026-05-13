package tui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// inFilter reports whether l is currently capturing keystrokes for filter
// typing. Use to skip the screen's custom keymap.
func inFilter(l list.Model) bool {
	return l.FilterState() == list.Filtering
}

// kindGlyph returns the single-cell symbol used to represent an OSM element
// kind in lists and headers: node = ●, way = ━, relation = ⬡.
func kindGlyph(kind string) string {
	switch kind {
	case "node":
		return "●"
	case "way":
		return "━"
	case "relation":
		return "⬡"
	}
	return "?"
}

// kindLabel returns the glyph followed by the kind name. Use in detail views
// and headers so a user wondering what a glyph means can find it spelled out.
func kindLabel(kind string) string {
	if kind == "" {
		return "?"
	}
	return kindGlyph(kind) + " " + kind
}

// newCompactDelegate returns a list delegate that renders each item on a
// single line: title only, no description, no inter-item spacing.
func newCompactDelegate() list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	return d
}

var (
	headerStyle     = lipgloss.NewStyle().Bold(true)
	footerStyle     = lipgloss.NewStyle().Faint(true).MarginTop(1)
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	mutedStyle      = lipgloss.NewStyle().Faint(true)
	paneFocused     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("12"))
	paneUnfocused   = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("8"))
	breadcrumbStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	tagKeyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	tagValueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
)

// styledTag returns "key = value" with subtle colour: cyan key, dim equals,
// green value. Use everywhere a single OSM tag is displayed.
func styledTag(key, value string) string {
	return tagKeyStyle.Render(key) + mutedStyle.Render(" = ") + tagValueStyle.Render(value)
}

// wrapText soft-wraps s to width columns, preserving paragraphs.
func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	return lipgloss.NewStyle().Width(width).Render(s)
}
