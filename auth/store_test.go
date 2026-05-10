package auth

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestTokenRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OSM_TOKEN_PATH", filepath.Join(dir, "token.json"))

	if _, err := LoadToken(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected fs.ErrNotExist, got %v", err)
	}

	in := &oauth2.Token{
		AccessToken:  "ak",
		TokenType:    "Bearer",
		RefreshToken: "rt",
		Expiry:       time.Now().Add(time.Hour).Truncate(time.Second).UTC(),
	}
	if err := SaveToken(in); err != nil {
		t.Fatalf("save: %v", err)
	}
	out, err := LoadToken()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if out.AccessToken != in.AccessToken || out.RefreshToken != in.RefreshToken {
		t.Errorf("token mismatch: got %+v want %+v", out, in)
	}
}
