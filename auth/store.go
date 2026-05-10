package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// tokenPath honours $OSM_TOKEN_PATH so tests can redirect persistence.
func tokenPath() (string, error) {
	if p := os.Getenv("OSM_TOKEN_PATH"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("user config dir: %w", err)
	}
	return filepath.Join(dir, "osm", "token.json"), nil
}

// LoadToken reads the persisted token. Returns fs.ErrNotExist if absent.
func LoadToken() (*oauth2.Token, error) {
	p, err := tokenPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fs.ErrNotExist
		}
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	var t oauth2.Token
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	return &t, nil
}

// RemoveToken deletes the persisted token; missing file is not an error.
func RemoveToken() error {
	p, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// SaveToken writes the token with mode 0600, creating parents with 0700.
func SaveToken(tok *oauth2.Token) error {
	p, err := tokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(p), err)
	}
	b, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}
	if err := os.WriteFile(p, b, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", p, err)
	}
	return nil
}

// HTTPClient returns an http.Client that refreshes the token automatically and
// writes any refreshed token back to disk.
func HTTPClient(ctx context.Context, cfg Config, tok *oauth2.Token) *http.Client {
	src := cfg.oauth2Config().TokenSource(ctx, tok)
	return oauth2.NewClient(ctx, &persistingSource{src: src, last: tok})
}

type persistingSource struct {
	src  oauth2.TokenSource
	last *oauth2.Token
}

func (p *persistingSource) Token() (*oauth2.Token, error) {
	t, err := p.src.Token()
	if err != nil {
		return nil, err
	}
	if p.last == nil || t.AccessToken != p.last.AccessToken {
		// Persistence is best-effort: a disk failure shouldn't break the request.
		_ = SaveToken(t)
		p.last = t
	}
	return t, nil
}
