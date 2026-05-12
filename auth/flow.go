// Package auth handles OAuth2 PKCE login against OpenStreetMap and persistence
// of the resulting token.
package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

const (
	DefaultAuthURL     = "https://www.openstreetmap.org/oauth2/authorize"
	DefaultTokenURL    = "https://www.openstreetmap.org/oauth2/token"
	DefaultRedirectURI = "http://127.0.0.1:17654/callback"
	LoginTimeout       = 5 * time.Minute
)

type Config struct {
	ClientID    string
	Scopes      []string
	AuthURL     string
	TokenURL    string
	RedirectURI string
	// OnAuthURL is called once with the authorization URL before the browser
	// is launched. Useful for headless environments where the URL must be
	// copied to a browser elsewhere.
	OnAuthURL func(url string)
}

// browser.OpenURL writes to its own stderr on some platforms; silence it.
func init() {
	browser.Stderr = io.Discard
	browser.Stdout = io.Discard
}

var openURL = browser.OpenURL

func (c Config) oauth2Config() *oauth2.Config {
	a := c.AuthURL
	if a == "" {
		a = DefaultAuthURL
	}
	t := c.TokenURL
	if t == "" {
		t = DefaultTokenURL
	}
	r := c.RedirectURI
	if r == "" {
		r = DefaultRedirectURI
	}
	return &oauth2.Config{
		ClientID:    c.ClientID,
		Endpoint:    oauth2.Endpoint{AuthURL: a, TokenURL: t, AuthStyle: oauth2.AuthStyleInParams},
		RedirectURL: r,
		Scopes:      c.Scopes,
	}
}

// Login runs the OAuth2 Authorization Code + PKCE flow and returns the token.
// Blocks until the browser handshake completes or LoginTimeout elapses.
func Login(ctx context.Context, cfg Config) (*oauth2.Token, error) {
	oc := cfg.oauth2Config()
	redirect, err := url.Parse(oc.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("parse redirect uri: %w", err)
	}
	if h := redirect.Hostname(); h != "127.0.0.1" && h != "localhost" {
		return nil, fmt.Errorf("redirect uri must be loopback, got %s", redirect.Host)
	}

	listener, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", redirect.Host, err)
	}
	defer listener.Close()

	// If the configured port was 0, replace it with the actual one before sending it to authorize.
	if redirect.Port() == "0" || redirect.Port() == "" {
		redirect.Host = listener.Addr().String()
		oc.RedirectURL = redirect.String()
	}

	verifier := oauth2.GenerateVerifier()
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	type result struct {
		code string
		err  error
	}
	out := make(chan result, 1)

	// deliver sends r to out exactly once; later callbacks are ignored so a
	// replay or double-load can't deadlock on the size-1 channel.
	deliver := func(r result) {
		select {
		case out <- r:
		default:
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc(redirect.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if e := q.Get("error"); e != "" {
			fmt.Fprintf(w, "login failed: %s\n%s", e, q.Get("error_description"))
			deliver(result{err: fmt.Errorf("authorize error %q: %s", e, q.Get("error_description"))})
			return
		}
		if q.Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			deliver(result{err: errors.New("oauth state mismatch")})
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			deliver(result{err: errors.New("missing authorization code")})
			return
		}
		fmt.Fprintln(w, "logged in. you can close this tab.")
		deliver(result{code: code})
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			deliver(result{err: fmt.Errorf("serve callback: %w", err)})
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	authURL := oc.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	if cfg.OnAuthURL != nil {
		cfg.OnAuthURL(authURL)
	}
	// Browser launch is best-effort: headless callers rely on OnAuthURL.
	_ = openURL(authURL)

	ctx, cancel := context.WithTimeout(ctx, LoginTimeout)
	defer cancel()

	var got result
	select {
	case got = <-out:
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, fmt.Errorf("login cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("login timed out after %s", LoginTimeout)
	}
	if got.err != nil {
		return nil, got.err
	}

	tok, err := oc.Exchange(ctx, got.code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	return tok, nil
}
