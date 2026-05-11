package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "show server capabilities and current token permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		caps, err := c.Capabilities(cmd.Context())
		if err != nil {
			return fmt.Errorf("capabilities: %w", err)
		}
		fmt.Println("server:")
		fmt.Printf("  api version       %s..%s\n", caps.MinAPIVersion, caps.MaxAPIVersion)
		fmt.Printf("  status            db=%s api=%s gpx=%s\n", caps.DatabaseStatus, caps.APIStatus, caps.GPXStatus)
		fmt.Printf("  area max          %g deg^2\n", caps.AreaMax)
		fmt.Printf("  note area max     %g\n", caps.NoteAreaMax)
		fmt.Printf("  changeset max     %d elements, query limit %d\n", caps.ChangesetMaxElements, caps.ChangesetMaxQuery)
		fmt.Printf("  way nodes max     %d\n", caps.WayNodesMax)
		fmt.Printf("  relation members  %d max\n", caps.RelationMembersMax)
		fmt.Printf("  notes query max   %d\n", caps.NotesMaxQuery)
		fmt.Printf("  tracepoints page  %d\n", caps.TracepointsPerPage)
		fmt.Printf("  timeout           %d s\n", caps.TimeoutSeconds)

		perms, err := c.Permissions(cmd.Context())
		if err != nil {
			return fmt.Errorf("permissions: %w", err)
		}
		fmt.Println("\ntoken permissions:")
		for _, p := range perms {
			fmt.Printf("  %s\n", p)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
