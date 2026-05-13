package cmd

import (
	"fmt"
	"net/url"
	"os"

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
		cfg.OnAuthURL = printAuthInstructions
		tok, err := auth.Login(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		if err := auth.SaveToken(tok); err != nil {
			return err
		}
		if err := auth.SaveConfig(&auth.PersistedConfig{ClientID: cfg.ClientID}); err != nil {
			fmt.Fprintf(os.Stderr, "warn: persist client id: %v\n", err)
		}
		fmt.Println("logged in")
		return nil
	},
}

func printAuthInstructions(u string) {
	port := callbackPort(u)
	fmt.Fprintln(os.Stderr, "open this url in a browser to authorize:")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "   ", u)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "if you are on a headless host, forward the callback port over ssh from your local machine first:")
	fmt.Fprintf(os.Stderr, "    ssh -L %s:127.0.0.1:%s <this-host>\n", port, port)
	fmt.Fprintln(os.Stderr, "waiting for callback (up to 5 minutes)...")
}

// callbackPort extracts the port from the authorize URL's redirect_uri.
// Falls back to "17654" (the default) if anything goes wrong.
func callbackPort(authURL string) string {
	u, err := url.Parse(authURL)
	if err != nil {
		return "17654"
	}
	redirect, err := url.Parse(u.Query().Get("redirect_uri"))
	if err != nil || redirect.Port() == "" {
		return "17654"
	}
	return redirect.Port()
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
