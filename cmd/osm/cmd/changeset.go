package cmd

import (
	"fmt"
	"os"
	"strconv"
	"text/template"
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
	csListFormat  string
	csOpenComment string
	csOpenSource  string
)

const defaultChangesetListFormat = "{{.ID}}\t{{.CreatedAt.Format \"2006-01-02\"}}\t{{.User}}\t{{.Comment}}"

var csListCmd = &cobra.Command{
	Use:   "list",
	Short: "list changesets",
	Long: `List changesets.

The --format flag accepts a Go text/template against each changeset. Fields
(from github.com/paulmach/osm.Changeset):
  ID, User, UserID, CreatedAt, ClosedAt, Open, ChangesCount, MinLat, MaxLat,
  MinLon, MaxLon, CommentsCount, Tags.
Methods callable from the template:
  Comment, Source, CreatedBy, Locale, ImageryUsed, Host, Bot.

CreatedAt and ClosedAt are time.Time; format them with .Format, e.g.
{{.CreatedAt.Format "2006-01-02 15:04"}}.

Example:
  osm changeset list --mine --format '{{.ID}} ({{.ChangesCount}} edits) {{.Comment}}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format := csListFormat
		if format == "" {
			format = defaultChangesetListFormat
		}
		tmpl, err := template.New("changesets").Parse(format)
		if err != nil {
			return fmt.Errorf("parse --format: %w", err)
		}
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
			if err := tmpl.Execute(os.Stdout, cs); err != nil {
				return err
			}
			fmt.Println()
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
	csListCmd.Flags().StringVar(&csListFormat, "format", "", "Go template per changeset (see --help for fields)")
	csOpenCmd.Flags().StringVar(&csOpenComment, "comment", "", "changeset comment tag")
	csOpenCmd.Flags().StringVar(&csOpenSource, "source", "", "changeset source tag")
	changesetCmd.AddCommand(csListCmd, csShowCmd, csOpenCmd, csCloseCmd, csCommentCmd)
	rootCmd.AddCommand(changesetCmd)
}
