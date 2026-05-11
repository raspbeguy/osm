package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	footerStyle = lipgloss.NewStyle().Faint(true).MarginTop(1)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	mutedStyle  = lipgloss.NewStyle().Faint(true)
)
