package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
)

// markdownStyle picks a glamour style without querying the terminal (which
// blocks for ~30 s over SSH waiting for an OSC reply). Override with
// GLAMOUR_STYLE.
func markdownStyle() string {
	if s := os.Getenv("GLAMOUR_STYLE"); s != "" {
		return s
	}
	return "dark"
}

// renderMarkdown returns s rendered as terminal-styled markdown, wrapped to
// width. Falls back to the raw input on any error.
func renderMarkdown(s string, width int) string {
	if s == "" {
		return ""
	}
	if width <= 0 {
		return s
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(markdownStyle()),
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
