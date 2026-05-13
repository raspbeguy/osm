//go:build !notui

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/raspbeguy/osm/cmd/osmctl/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui [target] [args...]",
	Short: "open the interactive terminal ui",
	Long: `Open the interactive TUI. The optional target deep-links to a screen
on launch:

  osmctl tui                       main menu
  osmctl tui profile               your osm profile
  osmctl tui inbox | outbox        messages
  osmctl tui changesets            your changesets list
  osmctl tui changeset <id>        one changeset
  osmctl tui notes                 notes lookup
  osmctl tui history <kind> <id>   element version history (kind = node|way|relation)
  osmctl tui doctor                server capabilities and token scopes
  osmctl tui traces                gps traces list
  osmctl tui compose | new         new changeset compose`,
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
