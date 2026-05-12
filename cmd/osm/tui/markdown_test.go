package tui

import "testing"

func TestIsLightBackground(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"15;0", false},   // dark
		{"0;15", true},    // light
		{"15;default", false},
		{"0;7", true},
		{"15;8", false},
		{"15;default;0", false},
		{"0;default;15", true},
	}
	for _, c := range cases {
		if got := isLightBackground(c.in); got != c.want {
			t.Errorf("%q: got %v, want %v", c.in, got, c.want)
		}
	}
}
