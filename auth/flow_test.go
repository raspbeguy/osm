package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
)

func newOAuthMock(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("code_challenge") == "" || q.Get("code_challenge_method") != "S256" {
			http.Error(w, "missing pkce challenge", http.StatusBadRequest)
			return
		}
		u, _ := url.Parse(q.Get("redirect_uri"))
		qq := u.Query()
		qq.Set("code", "test-code")
		qq.Set("state", q.Get("state"))
		u.RawQuery = qq.Encode()
		http.Redirect(w, r, u.String(), http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch {
		case r.Form.Get("grant_type") != "authorization_code":
			http.Error(w, "bad grant_type", http.StatusBadRequest)
		case r.Form.Get("code") != "test-code":
			http.Error(w, "bad code", http.StatusBadRequest)
		case r.Form.Get("code_verifier") == "":
			http.Error(w, "missing verifier", http.StatusBadRequest)
		default:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "tok",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		}
	})
	return httptest.NewServer(mux)
}

func TestLoginHappyPath(t *testing.T) {
	server := newOAuthMock(t)
	defer server.Close()

	t.Setenv("OSM_TOKEN_PATH", filepath.Join(t.TempDir(), "tok.json"))

	prev := openURL
	t.Cleanup(func() { openURL = prev })
	openURL = func(u string) error {
		resp, err := http.Get(u)
		if err != nil {
			return err
		}
		resp.Body.Close()
		return nil
	}

	tok, err := Login(context.Background(), Config{
		ClientID:    "test-client",
		Scopes:      []string{"read_prefs"},
		AuthURL:     server.URL + "/authorize",
		TokenURL:    server.URL + "/token",
		RedirectURI: "http://127.0.0.1:0/callback",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tok.AccessToken != "tok" {
		t.Errorf("got access_token %q, want %q", tok.AccessToken, "tok")
	}
}

func TestLoginRejectsNonLoopback(t *testing.T) {
	_, err := Login(context.Background(), Config{
		ClientID:    "x",
		RedirectURI: "http://example.com/cb",
	})
	if err == nil {
		t.Fatal("expected error for non-loopback redirect")
	}
}
