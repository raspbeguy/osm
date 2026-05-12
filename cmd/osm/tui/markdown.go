package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
)

// renderMarkdown returns s rendered as terminal-styled markdown, wrapped to
// width. If anything goes wrong, falls back to the raw input.
func renderMarkdown(s string, width int) string {
	if s == "" {
		return ""
	}
	if width <= 0 {
		return s
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return s
	}
	out, err := r.Render(s)
	if err != nil {
		return s
	}
	return strings.TrimRight(out, "\n")
}
