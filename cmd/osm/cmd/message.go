package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var messageCmd = &cobra.Command{Use: "message", Short: "manage your inbox"}

var msgInboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "list inbox messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newAPIClient(cmd.Context())
		if err != nil {
			return err
		}
		msgs, err := c.ListInbox(cmd.Context())
		if err != nil {
			return err
		}
		for _, m := range msgs {
			marker := " "
			if !m.Read {
				marker = "*"
			}
			fmt.Printf("%s %d\t%s\t%s\n", marker, m.ID, m.FromUser, m.Title)
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
	messageCmd.AddCommand(msgInboxCmd, msgReadCmd, msgDeleteCmd)
	rootCmd.AddCommand(messageCmd)
}
