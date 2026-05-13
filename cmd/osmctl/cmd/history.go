package cmd

import (
	"fmt"
	"strconv"

	"github.com/paulmach/osm"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:   "history <node|way|relation> <id>",
	Short: "show every version of an element",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kind := args[0]
		id, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		switch kind {
		case "node":
			vs, err := c.NodeHistory(cmd.Context(), osm.NodeID(id))
			if err != nil {
				return err
			}
			for _, n := range vs {
				visible := "visible"
				if !n.Visible {
					visible = "deleted"
				}
				fmt.Printf("v%-3d  %s  cs=%d  %s  %s\n", n.Version, n.Timestamp.Format("2006-01-02T15:04:05Z"), n.ChangesetID, visible, n.User)
			}
		case "way":
			vs, err := c.WayHistory(cmd.Context(), osm.WayID(id))
			if err != nil {
				return err
			}
			for _, w := range vs {
				visible := "visible"
				if !w.Visible {
					visible = "deleted"
				}
				fmt.Printf("v%-3d  %s  cs=%d  %s  %s\n", w.Version, w.Timestamp.Format("2006-01-02T15:04:05Z"), w.ChangesetID, visible, w.User)
			}
		case "relation":
			vs, err := c.RelationHistory(cmd.Context(), osm.RelationID(id))
			if err != nil {
				return err
			}
			for _, r := range vs {
				visible := "visible"
				if !r.Visible {
					visible = "deleted"
				}
				fmt.Printf("v%-3d  %s  cs=%d  %s  %s\n", r.Version, r.Timestamp.Format("2006-01-02T15:04:05Z"), r.ChangesetID, visible, r.User)
			}
		default:
			return fmt.Errorf("unknown element kind %q (want node, way, or relation)", kind)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
}
