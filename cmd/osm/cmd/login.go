package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/raspbeguy/osm/auth"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "authenticate via oauth2 in the browser",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := authConfig()
		if err != nil {
			return err
		}
		tok, err := auth.Login(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		if err := auth.SaveToken(tok); err != nil {
			return err
		}
		fmt.Println("logged in")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "remove the stored access token",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.RemoveToken(); err != nil {
			return err
		}
		fmt.Println("logged out")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd, logoutCmd)
}
