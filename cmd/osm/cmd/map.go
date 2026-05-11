package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map <l,b,r,t>",
	Short: "download every feature in a bbox as osm xml",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bbox, err := parseBBox(args[0])
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		out, err := c.DownloadMap(cmd.Context(), bbox)
		if err != nil {
			return err
		}
		fmt.Print(out)
		return nil
	},
}

func parseBBox(s string) ([4]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return [4]float64{}, fmt.Errorf("bbox needs 4 comma-separated values: l,b,r,t")
	}
	var bb [4]float64
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return [4]float64{}, fmt.Errorf("bbox value %d (%q): %w", i+1, p, err)
		}
		bb[i] = v
	}
	return bb, nil
}

func init() {
	rootCmd.AddCommand(mapCmd)
}
