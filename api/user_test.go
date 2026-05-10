package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := NewClient(http.DefaultClient)
	c.BaseURL = srv.URL
	return c
}

func TestWhoami(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/details.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"user":{"id":42,"display_name":"alice","account_created":"2010-01-01T00:00:00Z","description":"hi","changesets":{"count":7},"languages":["en","fr"]}}`)
	}))
	u, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatalf("whoami: %v", err)
	}
	if u.ID != 42 || u.DisplayName != "alice" || u.ChangesetCount != 7 || len(u.Languages) != 2 {
		t.Errorf("got %+v", u)
	}
}

func TestPreferencesRoundtrip(t *testing.T) {
	store := map[string]string{"theme": "dark"}
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/user/preferences.json":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"preferences":{"theme":"`+store["theme"]+`"}}`)
		case r.Method == "PUT" && r.URL.Path == "/user/preferences/theme":
			b, _ := io.ReadAll(r.Body)
			store["theme"] = string(b)
		case r.Method == "DELETE" && r.URL.Path == "/user/preferences/theme":
			delete(store, "theme")
		default:
			http.NotFound(w, r)
		}
	}))
	prefs, err := c.Preferences(context.Background())
	if err != nil || prefs["theme"] != "dark" {
		t.Fatalf("initial prefs: %+v err=%v", prefs, err)
	}
	if err := c.SetPreference(context.Background(), "theme", "light"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if store["theme"] != "light" {
		t.Errorf("server saw %q, want light", store["theme"])
	}
	if err := c.DeletePreference(context.Background(), "theme"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := store["theme"]; ok {
		t.Errorf("expected key deleted, store=%+v", store)
	}
}

func TestErrorMapping(t *testing.T) {
	cases := []struct {
		code int
		body string
		want error
	}{
		{401, "noauth", ErrUnauthorized},
		{403, "", ErrForbidden},
		{404, "", ErrNotFound},
		{409, "version mismatch", ErrConflict},
		{409, "The changeset 1 was closed at 2020-01-01", ErrChangesetClosed},
		{410, "", ErrGone},
		{412, "", ErrPreconditionFailed},
		{429, "", ErrTooManyRequests},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, tc.body, tc.code)
			}))
			_, err := c.Whoami(context.Background())
			if !errors.Is(err, tc.want) {
				t.Errorf("got %v, want errors.Is(%v) = true", err, tc.want)
			}
		})
	}
}
