//go:build notui

package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui [target] [args...]",
	Short: "open the interactive terminal ui (disabled in this build)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("this build was compiled without TUI support (build tag: notui); rebuild without -tags notui to enable")
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
