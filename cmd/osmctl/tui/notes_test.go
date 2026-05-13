package tui

import "testing"

func TestParseNoteQuery(t *testing.T) {
	cases := []struct {
		in     string
		id     int64
		bbox   [4]float64
		isID   bool
		errSub string
	}{
		{"329125", 329125, [4]float64{}, true, ""},
		{"6.945,48.581,6.948,48.583", 0, [4]float64{6.945, 48.581, 6.948, 48.583}, false, ""},
		{" 1 ", 1, [4]float64{}, true, ""},
		{"", 0, [4]float64{}, false, "empty"},
		{"abc", 0, [4]float64{}, false, "note ID or bbox"},
		{"1,2,3", 0, [4]float64{}, false, "4 comma"},
		{"1,2,3,x", 0, [4]float64{}, false, "bbox value 4"},
	}
	for _, c := range cases {
		id, bbox, isID, err := parseNoteQuery(c.in)
		if c.errSub != "" {
			if err == nil || !contains(err.Error(), c.errSub) {
				t.Errorf("%q: expected err containing %q, got %v", c.in, c.errSub, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: unexpected err %v", c.in, err)
			continue
		}
		if id != c.id || bbox != c.bbox || isID != c.isID {
			t.Errorf("%q: got (id=%d bbox=%v isID=%v), want (id=%d bbox=%v isID=%v)",
				c.in, id, bbox, isID, c.id, c.bbox, c.isID)
		}
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
