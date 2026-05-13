//go:build integration

package api_test

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/paulmach/osm"

	"github.com/raspbeguy/osm/api"
	"github.com/raspbeguy/osm/auth"
)

// TestSandboxElementRoundtrip exercises create/modify/delete against a real
// OSM sandbox. Skipped unless OSM_API_URL and OSM_CLIENT_ID are set and a token
// has been saved (run `osmctl login` first).
func TestSandboxElementRoundtrip(t *testing.T) {
	baseURL := os.Getenv("OSM_API_URL")
	clientID := os.Getenv("OSM_CLIENT_ID")
	if baseURL == "" || clientID == "" {
		t.Skip("set OSM_API_URL and OSM_CLIENT_ID to run")
	}

	tok, err := auth.LoadToken()
	if err != nil {
		t.Fatalf("load token (run `osmctl login` first): %v", err)
	}

	authURL, tokenURL := deriveOAuthEndpoints(baseURL)
	cfg := auth.Config{
		ClientID: clientID,
		AuthURL:  authURL,
		TokenURL: tokenURL,
		Scopes:   []string{"write_api"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := api.NewClient(auth.HTTPClient(ctx, cfg, tok))
	c.BaseURL = baseURL

	csID, err := c.OpenChangeset(ctx, osm.Tags{
		{Key: "created_by", Value: "osmctl integration test"},
		{Key: "comment", Value: "element roundtrip"},
	})
	if err != nil {
		t.Fatalf("open changeset: %v", err)
	}
	t.Logf("changeset %d", csID)

	t.Cleanup(func() {
		if err := c.CloseChangeset(context.Background(), csID); err != nil {
			t.Logf("close changeset: %v", err)
		}
	})

	create := &osm.Change{
		Version:   "0.6",
		Generator: "osmctl integration test",
		Create: &osm.OSM{
			Nodes: osm.Nodes{{
				ID: -1, Version: 0, Lat: 48.581541, Lon: 6.945972, Visible: true,
				Tags: osm.Tags{
					{Key: "amenity", Value: "bench"},
					{Key: "note", Value: "temporary test node"},
				},
			}},
		},
	}
	diff, err := c.UploadChange(ctx, csID, create)
	if err != nil {
		t.Fatalf("upload create: %v", err)
	}
	newID, err := firstCreatedNodeID(diff)
	if err != nil {
		t.Fatalf("parse diff: %v\n%s", err, diff)
	}
	t.Logf("created node %d", newID)

	n, err := c.GetNode(ctx, newID)
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	n.Tags = append(n.Tags, osm.Tag{Key: "test", Value: "tag-edit-roundtrip"})
	v2, err := c.ModifyNode(ctx, csID, n)
	if err != nil {
		t.Fatalf("modify node: %v", err)
	}
	t.Logf("modified to version %d", v2)

	refetched, err := c.GetNode(ctx, newID)
	if err != nil {
		t.Fatalf("get after modify: %v", err)
	}
	if refetched.Tags.Find("test") != "tag-edit-roundtrip" {
		t.Errorf("tag missing after modify, got: %v", refetched.Tags)
	}

	vDel, err := c.DeleteNode(ctx, csID, refetched)
	if err != nil {
		t.Fatalf("delete node: %v", err)
	}
	t.Logf("deleted, tombstone version %d", vDel)
}

func firstCreatedNodeID(diff string) (osm.NodeID, error) {
	var dr struct {
		Nodes []struct {
			OldID int64 `xml:"old_id,attr"`
			NewID int64 `xml:"new_id,attr"`
		} `xml:"node"`
	}
	if err := xml.Unmarshal([]byte(diff), &dr); err != nil {
		return 0, err
	}
	for _, n := range dr.Nodes {
		if n.OldID < 0 && n.NewID > 0 {
			return osm.NodeID(n.NewID), nil
		}
	}
	return 0, fmt.Errorf("no newly created node in diffResult")
}

// deriveOAuthEndpoints mirrors the CLI's host-derivation logic so the test
// reaches the OAuth server matching the API base.
func deriveOAuthEndpoints(base string) (string, string) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" {
		return auth.DefaultAuthURL, auth.DefaultTokenURL
	}
	host := u.Host
	if host == "api.openstreetmap.org" {
		host = "www.openstreetmap.org"
	}
	return "https://" + host + "/oauth2/authorize", "https://" + host + "/oauth2/token"
}
