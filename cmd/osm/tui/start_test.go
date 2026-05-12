package tui

import (
	"strings"
	"testing"
)

func TestStartAt(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		wantNil bool
		errSub  string
	}{
		{"empty", nil, true, ""},
		{"menu", []string{"menu"}, true, ""},
		{"profile", []string{"profile"}, false, ""},
		{"inbox", []string{"inbox"}, false, ""},
		{"changesets", []string{"changesets"}, false, ""},
		{"changeset by id", []string{"changeset", "12345"}, false, ""},
		{"changeset missing id", []string{"changeset"}, false, "usage"},
		{"changeset bad id", []string{"changeset", "abc"}, false, "invalid changeset id"},
		{"history full", []string{"history", "node", "42"}, false, ""},
		{"history wrong kind", []string{"history", "way2", "1"}, false, "kind must be"},
		{"history bad id", []string{"history", "node", "x"}, false, "invalid id"},
		{"unknown target", []string{"weird"}, false, "unknown target"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd, err := startAt(c.args)
			if c.errSub != "" {
				if err == nil || !strings.Contains(err.Error(), c.errSub) {
					t.Errorf("err = %v, want containing %q", err, c.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if c.wantNil && cmd != nil {
				t.Error("expected nil cmd")
			}
			if !c.wantNil && cmd == nil {
				t.Error("expected non-nil cmd")
			}
		})
	}
}
