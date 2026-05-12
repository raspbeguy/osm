// Package tui implements an interactive Bubbletea-based terminal UI for the
// osm CLI.
package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/raspbeguy/osm/api"
)

// programCtx is the context passed to Run. async tea.Cmd functions read it to
// cancel HTTP requests on program shutdown or OS signals. Safe because Run
// runs at most once per process.
var programCtx context.Context = context.Background()

// Run starts the TUI program against the given API client and returns when the
// user quits or the context is cancelled.
func Run(ctx context.Context, c *api.Client) error {
	programCtx = ctx
	p := tea.NewProgram(newRoot(c), tea.WithContext(ctx), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
