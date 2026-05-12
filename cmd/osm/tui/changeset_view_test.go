package tui

import (
	"strings"
	"testing"

	"github.com/paulmach/osm"
)

func TestFormatTagDiff(t *testing.T) {
	cur := osm.Tags{
		{Key: "amenity", Value: "restaurant"},
		{Key: "cuisine", Value: "italian"},
	}
	prev := osm.Tags{
		{Key: "amenity", Value: "cafe"},
		{Key: "phone", Value: "+33 1 23 45"},
	}
	got := formatTagDiff(cur, prev)
	for _, want := range []string{
		"~ amenity: cafe → restaurant",
		"+ cuisine = italian",
		"- phone = +33 1 23 45",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("diff missing %q\n%s", want, got)
		}
	}
}

func TestFormatTagDiffNoChange(t *testing.T) {
	t1 := osm.Tags{{Key: "a", Value: "b"}}
	if got := formatTagDiff(t1, t1); got != "" {
		t.Errorf("expected empty diff, got %q", got)
	}
}

func TestNonTagChanged(t *testing.T) {
	tests := []struct {
		name string
		cur  changesetElement
		prev *prevElement
		want bool
	}{
		{"node geometry equal", changesetElement{Kind: "node", Lat: 1, Lon: 2}, &prevElement{Lat: 1, Lon: 2}, false},
		{"node geometry moved", changesetElement{Kind: "node", Lat: 1, Lon: 2}, &prevElement{Lat: 1.5, Lon: 2}, true},
		{"way refs equal", changesetElement{Kind: "way", Nodes: []int64{1, 2, 3}}, &prevElement{Nodes: []int64{1, 2, 3}}, false},
		{"way refs reordered", changesetElement{Kind: "way", Nodes: []int64{1, 2, 3}}, &prevElement{Nodes: []int64{1, 3, 2}}, true},
		{"way refs different length", changesetElement{Kind: "way", Nodes: []int64{1, 2}}, &prevElement{Nodes: []int64{1, 2, 3}}, true},
		{"relation members equal", changesetElement{Kind: "relation", Members: []memberDescr{{Type: "node", Ref: 1, Role: "outer"}}}, &prevElement{Members: []memberDescr{{Type: "node", Ref: 1, Role: "outer"}}}, false},
		{"relation member role changed", changesetElement{Kind: "relation", Members: []memberDescr{{Type: "node", Ref: 1, Role: "inner"}}}, &prevElement{Members: []memberDescr{{Type: "node", Ref: 1, Role: "outer"}}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nonTagChanged(tc.cur, tc.prev); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
