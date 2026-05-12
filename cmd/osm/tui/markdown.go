package tui

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
)

// markdownStyle picks a glamour style without doing an OSC terminal probe
// (which blocks for ~30 s over SSH). Honours GLAMOUR_STYLE if set, otherwise
// reads COLORFGBG as terminals like xterm/Konsole/urxvt provide it, and falls
// back to "dark".
func markdownStyle() string {
	if s := os.Getenv("GLAMOUR_STYLE"); s != "" {
		return s
	}
	if isLightBackground(os.Getenv("COLORFGBG")) {
		return "light"
	}
	return "dark"
}

// isLightBackground parses the last segment of a COLORFGBG value (e.g.
// "15;0", "0;15", "15;default;0") as an ANSI colour index. Indices 7 and
// 9..15 are the "light" half of the basic 16-colour palette.
func isLightBackground(v string) bool {
	if v == "" {
		return false
	}
	parts := strings.Split(v, ";")
	bg := parts[len(parts)-1]
	n, err := strconv.Atoi(bg)
	if err != nil {
		return false
	}
	return n == 7 || (n >= 9 && n <= 15)
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
