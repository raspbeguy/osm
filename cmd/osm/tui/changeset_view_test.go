package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/paulmach/osm"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestFormatTagDiff(t *testing.T) {
	cur := osm.Tags{
		{Key: "amenity", Value: "restaurant"},
		{Key: "cuisine", Value: "italian"},
	}
	prev := osm.Tags{
		{Key: "amenity", Value: "cafe"},
		{Key: "phone", Value: "+33 1 23 45"},
	}
	got := stripANSI(formatTagDiff(cur, prev))
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
	if got := stripANSI(formatTagDiff(t1, t1)); got != "" {
		t.Errorf("expected empty diff, got %q", got)
	}
}

func TestExtractChangesetElements(t *testing.T) {
	in := `<?xml version="1.0"?>
<osmChange version="0.6">
  <create>
    <node id="-1" version="0" lat="48.5" lon="6.9">
      <tag k="amenity" v="pharmacy"/>
    </node>
    <way id="-2" version="0">
      <nd ref="-1"/>
      <nd ref="123"/>
      <tag k="highway" v="footway"/>
    </way>
  </create>
  <modify>
    <relation id="99" version="3">
      <member type="way" ref="-2" role="outer"/>
      <tag k="type" v="multipolygon"/>
    </relation>
  </modify>
  <delete>
    <node id="555" version="2"/>
  </delete>
</osmChange>`
	elems, err := extractChangesetElements(in)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(elems) != 4 {
		t.Fatalf("got %d elements, want 4", len(elems))
	}
	if elems[0].Kind != "node" || elems[0].Action != '+' || elems[0].Lat != 48.5 {
		t.Errorf("first: %+v", elems[0])
	}
	if elems[1].Kind != "way" || len(elems[1].Nodes) != 2 || elems[1].Nodes[0] != -1 {
		t.Errorf("way: %+v", elems[1])
	}
	if elems[2].Kind != "relation" || elems[2].Action != '~' || len(elems[2].Members) != 1 || elems[2].Members[0].Role != "outer" {
		t.Errorf("rel: %+v", elems[2])
	}
	if elems[3].Action != '-' || elems[3].ID != 555 {
		t.Errorf("delete: %+v", elems[3])
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
