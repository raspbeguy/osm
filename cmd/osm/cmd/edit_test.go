package cmd

import (
	"testing"

	"github.com/paulmach/osm"
)

func TestMergeTagsAdd(t *testing.T) {
	dst := osm.Tags{{Key: "amenity", Value: "cafe"}}
	mergeTags(&dst, osm.Tags{{Key: "cuisine", Value: "italian"}})
	if dst.Find("cuisine") != "italian" {
		t.Errorf("add: %+v", dst)
	}
	if dst.Find("amenity") != "cafe" {
		t.Errorf("existing lost: %+v", dst)
	}
}

func TestMergeTagsOverwrite(t *testing.T) {
	dst := osm.Tags{{Key: "amenity", Value: "cafe"}}
	mergeTags(&dst, osm.Tags{{Key: "amenity", Value: "restaurant"}})
	if dst.Find("amenity") != "restaurant" || len(dst) != 1 {
		t.Errorf("overwrite: %+v", dst)
	}
}

func TestMergeTagsDeleteEmptyValue(t *testing.T) {
	dst := osm.Tags{{Key: "amenity", Value: "cafe"}, {Key: "name", Value: "x"}}
	mergeTags(&dst, osm.Tags{{Key: "amenity", Value: ""}})
	if dst.Find("amenity") != "" || dst.HasTag("amenity") {
		t.Errorf("delete: %+v", dst)
	}
	if dst.Find("name") != "x" {
		t.Errorf("other tag lost: %+v", dst)
	}
}

func TestMergeTagsDeleteMissing(t *testing.T) {
	dst := osm.Tags{{Key: "amenity", Value: "cafe"}}
	mergeTags(&dst, osm.Tags{{Key: "absent", Value: ""}})
	if len(dst) != 1 || dst.Find("amenity") != "cafe" {
		t.Errorf("unexpected: %+v", dst)
	}
}
