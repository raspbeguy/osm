package tui

import (
	"bytes"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// highlightXML returns s with ANSI colour escapes for XML syntax. Returns the
// raw input on any error so the viewport never goes blank.
func highlightXML(s string) string {
	if s == "" {
		return s
	}
	lexer := lexers.Get("xml")
	if lexer == nil {
		return s
	}
	style := styles.Get(chromaStyle())
	if style == nil {
		style = styles.Fallback
	}
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		return s
	}
	iter, err := lexer.Tokenise(nil, s)
	if err != nil {
		return s
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iter); err != nil {
		return s
	}
	return buf.String()
}

// chromaStyle picks a chroma palette tied to the terminal background, matching
// the same COLORFGBG / GLAMOUR_STYLE convention used for markdown.
func chromaStyle() string {
	if markdownStyle() == "light" {
		return "monokailight"
	}
	return "monokai"
}
