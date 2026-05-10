package cmd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/raspbeguy/osm/api"
	"github.com/raspbeguy/osm/auth"
)

var (
	flagAPI      string
	flagClientID string
)

var rootCmd = &cobra.Command{
	Use:           "osm",
	Short:         "command-line client for openstreetmap",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func ExecuteContext(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagAPI, "api", "", "osm api base url (or set OSM_API_URL)")
	rootCmd.PersistentFlags().StringVar(&flagClientID, "client-id", "", "oauth2 client id (or set OSM_CLIENT_ID)")
}

func apiBaseURL() string {
	if flagAPI != "" {
		return flagAPI
	}
	if e := os.Getenv("OSM_API_URL"); e != "" {
		return e
	}
	return api.DefaultBaseURL
}

// authBases derives the OAuth2 endpoints from the configured API base.
// Production and sandbox have distinct OAuth hosts.
func authBases() (string, string) {
	switch apiBaseURL() {
	case api.SandboxBaseURL:
		return "https://master.apis.dev.openstreetmap.org/oauth2/authorize",
			"https://master.apis.dev.openstreetmap.org/oauth2/token"
	default:
		return auth.DefaultAuthURL, auth.DefaultTokenURL
	}
}

func clientID() (string, error) {
	if flagClientID != "" {
		return flagClientID, nil
	}
	if e := os.Getenv("OSM_CLIENT_ID"); e != "" {
		return e, nil
	}
	return "", errors.New("no oauth2 client id; pass --client-id or set OSM_CLIENT_ID (register an app at https://www.openstreetmap.org/oauth2/applications)")
}

func authConfig() (auth.Config, error) {
	cid, err := clientID()
	if err != nil {
		return auth.Config{}, err
	}
	a, t := authBases()
	return auth.Config{
		ClientID: cid,
		Scopes: []string{
			"openid", "read_prefs", "write_prefs",
			"write_api", "write_notes", "consume_messages",
		},
		AuthURL:  a,
		TokenURL: t,
	}, nil
}

func newAPIClient(ctx context.Context) (*api.Client, error) {
	tok, err := auth.LoadToken()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("not logged in; run `osm login` first")
		}
		return nil, err
	}
	cfg, err := authConfig()
	if err != nil {
		return nil, err
	}
	c := api.NewClient(auth.HTTPClient(ctx, cfg, tok))
	c.BaseURL = apiBaseURL()
	return c, nil
}
