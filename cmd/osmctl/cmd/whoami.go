package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "print the authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		u, err := c.Whoami(cmd.Context())
		if err != nil {
			return err
		}
		fmt.Printf("%s (id=%d, %d changesets)\n", u.DisplayName, u.ID, u.ChangesetCount)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
