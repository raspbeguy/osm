package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/raspbeguy/osm/api"
)

var traceCmd = &cobra.Command{Use: "trace", Short: "manage your gps traces"}

var (
	traceUploadDesc       string
	traceUploadTags       string
	traceUploadVisibility string
	traceUpdateDesc       string
	traceUpdateTags       string
	traceUpdateVisibility string
)

var traceListCmd = &cobra.Command{
	Use:   "list",
	Short: "list your uploaded traces",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		traces, err := c.ListTraces(cmd.Context())
		if err != nil {
			return err
		}
		for _, t := range traces {
			date := t.Timestamp
			if len(date) >= 10 {
				date = date[:10]
			}
			fmt.Printf("%d\t%s\t%s\t%s\n", t.ID, t.Visibility, date, t.Description)
		}
		return nil
	},
}

var traceShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "show one trace's metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		t, err := c.GetTrace(cmd.Context(), id)
		if err != nil {
			return err
		}
		fmt.Printf("ID: %d\nName: %s\nUser: %s\nVisibility: %s\nPending: %v\nTimestamp: %s\nLocation: %g, %g\nDescription: %s\nTags: %s\n",
			t.ID, t.Name, t.User, t.Visibility, t.Pending, t.Timestamp, t.Lat, t.Lon, t.Description, strings.Join(t.Tags, ", "))
		return nil
	},
}

var traceDataCmd = &cobra.Command{
	Use:   "data <id>",
	Short: "print the raw gpx file of a trace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		data, err := c.GetTraceData(cmd.Context(), id)
		if err != nil {
			return err
		}
		fmt.Print(data)
		return nil
	},
}

var traceUploadCmd = &cobra.Command{
	Use:   "upload <gpx-file>",
	Short: "upload a gpx file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		var tags []string
		if traceUploadTags != "" {
			tags = strings.Split(traceUploadTags, ",")
		}
		id, err := c.UploadTrace(cmd.Context(), api.TraceUpload{
			Filename:    filepath.Base(path),
			GPX:         data,
			Description: traceUploadDesc,
			Tags:        tags,
			Visibility:  traceUploadVisibility,
		})
		if err != nil {
			return err
		}
		fmt.Println(id)
		return nil
	},
}

var traceUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "update a trace's metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		var tags []string
		if traceUpdateTags != "" {
			tags = strings.Split(traceUpdateTags, ",")
		}
		return c.UpdateTrace(cmd.Context(), id, api.TraceUpdate{
			Description: traceUpdateDesc,
			Tags:        tags,
			Visibility:  traceUpdateVisibility,
		})
	},
}

var traceDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "delete a trace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		return c.DeleteTrace(cmd.Context(), id)
	},
}

func init() {
	traceUploadCmd.Flags().StringVar(&traceUploadDesc, "description", "", "trace description (required)")
	traceUploadCmd.Flags().StringVar(&traceUploadTags, "tags", "", "comma-separated tags")
	traceUploadCmd.Flags().StringVar(&traceUploadVisibility, "visibility", "private", "public|trackable|identifiable|private")
	_ = traceUploadCmd.MarkFlagRequired("description")

	traceUpdateCmd.Flags().StringVar(&traceUpdateDesc, "description", "", "new description")
	traceUpdateCmd.Flags().StringVar(&traceUpdateTags, "tags", "", "new comma-separated tags")
	traceUpdateCmd.Flags().StringVar(&traceUpdateVisibility, "visibility", "", "public|trackable|identifiable|private")

	traceCmd.AddCommand(traceListCmd, traceShowCmd, traceDataCmd, traceUploadCmd, traceUpdateCmd, traceDeleteCmd)
	rootCmd.AddCommand(traceCmd)
}
