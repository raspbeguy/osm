package tui

import (
	"testing"

	"github.com/paulmach/osm"
)

func TestBuildChange(t *testing.T) {
	staged := []*stagedElement{
		{Kind: "node", ID: 1, Version: 3, Action: stagedModify, Lat: 1, Lon: 2, Tags: osm.Tags{{Key: "a", Value: "b"}}},
		{Kind: "relation", ID: -1, Action: stagedCreate, Tags: osm.Tags{{Key: "type", Value: "route"}}},
	}
	ch := buildChange(staged)
	if ch.Modify == nil || len(ch.Modify.Nodes) != 1 {
		t.Fatalf("modify nodes wrong: %+v", ch.Modify)
	}
	if ch.Modify.Nodes[0].Tags.Find("a") != "b" {
		t.Errorf("modify tags lost: %+v", ch.Modify.Nodes[0])
	}
	if ch.Create == nil || len(ch.Create.Relations) != 1 || ch.Create.Relations[0].Tags.Find("type") != "route" {
		t.Errorf("create rel wrong: %+v", ch.Create)
	}
	if ch.Create != nil && len(ch.Create.Nodes) != 0 {
		t.Errorf("create nodes should be empty: %+v", ch.Create.Nodes)
	}
}

func TestBuildChangeEmpty(t *testing.T) {
	ch := buildChange(nil)
	if ch.Create != nil || ch.Modify != nil {
		t.Errorf("empty staged should produce no sections, got %+v", ch)
	}
	if ch.Version != "0.6" {
		t.Errorf("missing version: %+v", ch)
	}
}

func TestCloneStaged(t *testing.T) {
	original := &stagedElement{
		Kind: "way", ID: 1, Tags: osm.Tags{{Key: "k", Value: "v"}},
		Nodes: osm.WayNodes{{ID: 10}, {ID: 20}},
	}
	clone := cloneStaged([]*stagedElement{original})
	if clone[0] == original {
		t.Fatal("did not deep-copy: same pointer")
	}
	clone[0].Tags[0].Value = "mutated"
	if original.Tags[0].Value == "mutated" {
		t.Error("tag slice not deep-copied")
	}
	clone[0].Nodes[0].ID = 999
	if original.Nodes[0].ID == 999 {
		t.Error("node slice not deep-copied")
	}
}
