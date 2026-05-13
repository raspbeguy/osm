package tui

import (
	"strings"
	"testing"
)

func TestParseHistoryQuery(t *testing.T) {
	cases := []struct {
		in       string
		wantKind string
		wantID   int64
		errSub   string
	}{
		{"node 123", "node", 123, ""},
		{"way 456", "way", 456, ""},
		{"relation 789", "relation", 789, ""},
		{"  node   42  ", "node", 42, ""},
		{"", "", 0, "expected"},
		{"node", "", 0, "expected"},
		{"node 1 extra", "", 0, "expected"},
		{"foo 1", "", 0, "kind must be"},
		{"node abc", "", 0, "not a numeric"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			kind, id, err := parseHistoryQuery(c.in)
			if c.errSub != "" {
				if err == nil || !strings.Contains(err.Error(), c.errSub) {
					t.Errorf("err = %v, want containing %q", err, c.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if kind != c.wantKind || id != c.wantID {
				t.Errorf("got (%q, %d), want (%q, %d)", kind, id, c.wantKind, c.wantID)
			}
		})
	}
}
