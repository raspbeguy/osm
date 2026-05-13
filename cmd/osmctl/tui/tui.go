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

// Run starts the TUI program. args is the optional deep-link target (see
// startAt) that decides which screen to open on launch.
func Run(ctx context.Context, c *api.Client, args []string) error {
	startCmd, err := startAt(args)
	if err != nil {
		return err
	}
	programCtx = ctx
	p := tea.NewProgram(newRoot(c, startCmd), tea.WithContext(ctx), tea.WithAltScreen())
	_, err = p.Run()
	return err
}
