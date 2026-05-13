//go:build !notui

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/raspbeguy/osm/cmd/osm/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui [target] [args...]",
	Short: "open the interactive terminal ui",
	Long: `Open the interactive TUI. The optional target deep-links to a screen
on launch:

  osm tui                       main menu
  osm tui profile               your osm profile
  osm tui inbox | outbox        messages
  osm tui changesets            your changesets list
  osm tui changeset <id>        one changeset
  osm tui notes                 notes lookup
  osm tui history <kind> <id>   element version history (kind = node|way|relation)
  osm tui doctor                server capabilities and token scopes
  osm tui traces                gps traces list
  osm tui compose | new         new changeset compose`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		return tui.Run(cmd.Context(), c, args)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
