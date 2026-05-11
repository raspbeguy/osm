// Package tui implements an interactive Bubbletea-based terminal UI for the
// osm CLI.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

// Run starts the TUI program against the given API client and returns when the
// user quits or the context is cancelled.
func Run(ctx context.Context, c *api.Client) error {
	p := tea.NewProgram(newRoot(c), tea.WithContext(ctx), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
