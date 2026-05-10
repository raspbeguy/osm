package api

import (
	"context"
	"errors"
	"net/url"
)

type User struct {
	ID             int64
	DisplayName    string
	AccountCreated string
	Description    string
	ChangesetCount int
	Languages      []string
}

// Whoami returns the authenticated user via /user/details.
func (c *Client) Whoami(ctx context.Context) (*User, error) {
	var wrap struct {
		User struct {
			ID             int64  `json:"id"`
			DisplayName    string `json:"display_name"`
			AccountCreated string `json:"account_created"`
			Description    string `json:"description"`
			Changesets     struct {
				Count int `json:"count"`
			} `json:"changesets"`
			Languages []string `json:"languages"`
		} `json:"user"`
	}
	if err := c.getJSON(ctx, "/user/details.json", &wrap); err != nil {
		return nil, err
	}
	u := wrap.User
	return &User{
		ID:             u.ID,
		DisplayName:    u.DisplayName,
		AccountCreated: u.AccountCreated,
		Description:    u.Description,
		ChangesetCount: u.Changesets.Count,
		Languages:      u.Languages,
	}, nil
}

// Preferences returns the authenticated user's preference key/value map.
func (c *Client) Preferences(ctx context.Context) (map[string]string, error) {
	var wrap struct {
		Preferences map[string]string `json:"preferences"`
	}
	if err := c.getJSON(ctx, "/user/preferences.json", &wrap); err != nil {
		return nil, err
	}
	if wrap.Preferences == nil {
		wrap.Preferences = map[string]string{}
	}
	return wrap.Preferences, nil
}

// SetPreference upserts a single preference.
func (c *Client) SetPreference(ctx context.Context, key, value string) error {
	if key == "" {
		return errors.New("preference key required")
	}
	_, err := c.sendBody(ctx, "PUT", "/user/preferences/"+url.PathEscape(key), []byte(value), "text/plain")
	return err
}

// DeletePreference removes a single preference.
func (c *Client) DeletePreference(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("preference key required")
	}
	_, err := c.sendBody(ctx, "DELETE", "/user/preferences/"+url.PathEscape(key), nil, "")
	return err
}
