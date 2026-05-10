package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/paulmach/osm"
	"github.com/spf13/cobra"

	osmapi "github.com/raspbeguy/osm/api"
)

var changesetCmd = &cobra.Command{
	Use:   "changeset",
	Short: "manage changesets",
}

var (
	csListMine    bool
	csOpenComment string
	csOpenSource  string
)

var csListCmd = &cobra.Command{
	Use:   "list",
	Short: "list changesets",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		f := osmapi.ChangesetFilter{}
		if csListMine {
			u, err := c.Whoami(cmd.Context())
			if err != nil {
				return fmt.Errorf("resolve current user: %w", err)
			}
			f.UserID = u.ID
		}
		css, err := c.ListChangesets(cmd.Context(), f)
		if err != nil {
			return err
		}
		for _, cs := range css {
			fmt.Printf("%d\t%s\t%s\t%s\n", cs.ID, cs.User, cs.CreatedAt.Format(time.RFC3339), cs.Comment())
		}
		return nil
	},
}

var csShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "show a changeset",
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
		cs, err := c.GetChangeset(cmd.Context(), osm.ChangesetID(id))
		if err != nil {
			return err
		}
		fmt.Printf("id: %d\nuser: %s\nopen: %v\ncreated: %s\ncomment: %s\nsource: %s\n",
			cs.ID, cs.User, cs.Open, cs.CreatedAt.Format(time.RFC3339), cs.Comment(), cs.Source())
		return nil
	},
}

var csOpenCmd = &cobra.Command{
	Use:   "open",
	Short: "open a new changeset",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		tags := osm.Tags{{Key: "created_by", Value: "osm-go"}}
		if csOpenComment != "" {
			tags = append(tags, osm.Tag{Key: "comment", Value: csOpenComment})
		}
		if csOpenSource != "" {
			tags = append(tags, osm.Tag{Key: "source", Value: csOpenSource})
		}
		id, err := c.OpenChangeset(cmd.Context(), tags)
		if err != nil {
			return err
		}
		fmt.Println(int64(id))
		return nil
	},
}

var csCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "close an open changeset",
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
		return c.CloseChangeset(cmd.Context(), osm.ChangesetID(id))
	},
}

var csCommentCmd = &cobra.Command{
	Use:   "comment <id> <text>",
	Short: "post a comment on a changeset",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return err
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		return c.CommentChangeset(cmd.Context(), osm.ChangesetID(id), args[1])
	},
}

func init() {
	csListCmd.Flags().BoolVar(&csListMine, "mine", false, "only my changesets")
	csOpenCmd.Flags().StringVar(&csOpenComment, "comment", "", "changeset comment tag")
	csOpenCmd.Flags().StringVar(&csOpenSource, "source", "", "changeset source tag")
	changesetCmd.AddCommand(csListCmd, csShowCmd, csOpenCmd, csCloseCmd, csCommentCmd)
	rootCmd.AddCommand(changesetCmd)
}
