package osmchange

import (
	"strings"
	"testing"

	"github.com/paulmach/osm"
)

func TestMarshalStampsChangeset(t *testing.T) {
	ch := &osm.Change{
		Version:   "0.6",
		Generator: "test",
		Modify: &osm.OSM{
			Nodes: osm.Nodes{
				{ID: 42, Version: 3, Lat: 1, Lon: 2, Tags: osm.Tags{{Key: "amenity", Value: "bench"}}},
			},
		},
	}
	out, err := Marshal(osm.ChangesetID(99), ch)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "<osmChange") {
		t.Errorf("missing osmChange root: %s", s)
	}
	if !strings.Contains(s, "<modify>") {
		t.Errorf("missing <modify>: %s", s)
	}
	if !strings.Contains(s, `changeset="99"`) {
		t.Errorf("changeset attr not stamped: %s", s)
	}
	if !strings.Contains(s, `k="amenity"`) {
		t.Errorf("tag missing: %s", s)
	}
}

func TestMarshalNilChange(t *testing.T) {
	if _, err := Marshal(1, nil); err == nil {
		t.Fatal("expected error for nil change")
	}
}
