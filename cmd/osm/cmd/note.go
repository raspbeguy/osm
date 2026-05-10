package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{Use: "note", Short: "manage map notes"}

var (
	noteLat float64
	noteLon float64
)

var noteCreateCmd = &cobra.Command{
	Use:   "create <text>",
	Short: "create a map note",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		n, err := c.CreateNote(cmd.Context(), noteLat, noteLon, strings.Join(args, " "))
		if err != nil {
			return err
		}
		fmt.Println(n.ID)
		return nil
	},
}

var noteShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "show a map note",
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
		n, err := c.GetNote(cmd.Context(), id)
		if err != nil {
			return err
		}
		fmt.Printf("id: %d\nstatus: %s\nlat: %g\nlon: %g\ncreated: %s\n", n.ID, n.Status, n.Lat, n.Lon, n.CreatedAt)
		for _, ct := range n.Comments {
			fmt.Printf("- %s by %s: %s\n", ct.Action, ct.User, ct.Text)
		}
		return nil
	},
}

var noteCommentCmd = &cobra.Command{
	Use:   "comment <id> <text>",
	Short: "comment on a note",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		_, err = c.CommentNote(cmd.Context(), id, strings.Join(args[1:], " "))
		return err
	},
}

var noteCloseCmd = &cobra.Command{
	Use:   "close <id> [text]",
	Short: "close a note",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		text := ""
		if len(args) > 1 {
			text = strings.Join(args[1:], " ")
		}
		_, err = c.CloseNote(cmd.Context(), id, text)
		return err
	},
}

var noteReopenCmd = &cobra.Command{
	Use:   "reopen <id> [text]",
	Short: "reopen a closed note",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		text := ""
		if len(args) > 1 {
			text = strings.Join(args[1:], " ")
		}
		_, err = c.ReopenNote(cmd.Context(), id, text)
		return err
	},
}

func init() {
	noteCreateCmd.Flags().Float64Var(&noteLat, "lat", 0, "latitude")
	noteCreateCmd.Flags().Float64Var(&noteLon, "lon", 0, "longitude")
	_ = noteCreateCmd.MarkFlagRequired("lat")
	_ = noteCreateCmd.MarkFlagRequired("lon")
	noteCmd.AddCommand(noteCreateCmd, noteShowCmd, noteCommentCmd, noteCloseCmd, noteReopenCmd)
	rootCmd.AddCommand(noteCmd)
}
