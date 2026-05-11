package cmd

import (
	"github.com/spf13/cobra"

	"github.com/raspbeguy/osm/cmd/osm/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "open the interactive terminal ui",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		return tui.Run(cmd.Context(), c)
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
