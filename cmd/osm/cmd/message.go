package cmd

import (
	"fmt"
	"os"
	"strconv"
	"text/template"

	"github.com/spf13/cobra"
)

var messageCmd = &cobra.Command{Use: "message", Short: "manage your inbox"}

const defaultInboxFormat = "{{if .Read}} {{else}}*{{end}} {{.ID}}\t{{date .SentOn}}\t{{.FromUser}}\t{{.Title}}"

var inboxFormat string

var inboxFuncs = template.FuncMap{
	"date": func(s string) string {
		if len(s) >= 10 {
			return s[:10]
		}
		return s
	},
}

var msgInboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "list inbox messages",
	Long: `List inbox messages.

The --format flag accepts a Go text/template against each message. Fields:
  ID, FromUser, FromUserID, ToUser, ToUserID, SentOn, Title, BodyFormat,
  Read, Deleted. Body is not returned in listings; use 'osm message read' to
  fetch a single message including its body.

Template functions:
  date <s>   truncate an RFC3339 timestamp to "YYYY-MM-DD".

Example:
  osm message inbox --format '{{.SentOn}}  {{.FromUser}}: {{.Title}}'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format := inboxFormat
		if format == "" {
			format = defaultInboxFormat
		}
		tmpl, err := template.New("inbox").Funcs(inboxFuncs).Parse(format)
		if err != nil {
			return fmt.Errorf("parse --format: %w", err)
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		msgs, err := c.ListInbox(cmd.Context())
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if err := tmpl.Execute(os.Stdout, m); err != nil {
				return err
			}
			fmt.Println()
		}
		return nil
	},
}

var msgOutboxCmd = &cobra.Command{
	Use:   "outbox",
	Short: "list sent messages",
	Long:  msgInboxCmd.Long,
	RunE: func(cmd *cobra.Command, args []string) error {
		format := inboxFormat
		if format == "" {
			format = defaultInboxFormat
		}
		tmpl, err := template.New("outbox").Funcs(inboxFuncs).Parse(format)
		if err != nil {
			return fmt.Errorf("parse --format: %w", err)
		}
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		msgs, err := c.ListOutbox(cmd.Context())
		if err != nil {
			return err
		}
		for _, m := range msgs {
			if err := tmpl.Execute(os.Stdout, m); err != nil {
				return err
			}
			fmt.Println()
		}
		return nil
	},
}

var msgReadCmd = &cobra.Command{
	Use:   "read <id>",
	Short: "show a message",
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
		m, err := c.GetMessage(cmd.Context(), id)
		if err != nil {
			return err
		}
		fmt.Printf("from: %s\nsent: %s\nsubject: %s\n\n%s\n", m.FromUser, m.SentOn, m.Title, m.Body)
		return nil
	},
}

var msgDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "delete a message",
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
		return c.DeleteMessage(cmd.Context(), id)
	},
}

func init() {
	msgInboxCmd.Flags().StringVar(&inboxFormat, "format", "", "Go template per message (see --help for fields)")
	msgOutboxCmd.Flags().StringVar(&inboxFormat, "format", "", "Go template per message (see --help for fields)")
	messageCmd.AddCommand(msgInboxCmd, msgOutboxCmd, msgReadCmd, msgDeleteCmd)
	rootCmd.AddCommand(messageCmd)
}
