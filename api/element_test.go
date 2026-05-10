package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/paulmach/osm"
)

func TestGetNode(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/node/1" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, `<osm><node id="1" version="2" lat="1.0" lon="2.0" changeset="5" timestamp="2024-01-01T00:00:00Z" visible="true"><tag k="amenity" v="bench"/></node></osm>`)
	}))
	n, err := c.GetNode(context.Background(), 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if n.ID != 1 || n.Version != 2 || n.Tags.Find("amenity") != "bench" {
		t.Errorf("got %+v", n)
	}
}

func TestModifyNode(t *testing.T) {
	var seen string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/node/1" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
		io.WriteString(w, "3")
	}))
	n := &osm.Node{ID: 1, Version: 2, Lat: 1, Lon: 2, Tags: osm.Tags{{Key: "amenity", Value: "cafe"}}}
	v, err := c.ModifyNode(context.Background(), 5, n)
	if err != nil {
		t.Fatalf("modify: %v", err)
	}
	if v != 3 {
		t.Errorf("got version %d", v)
	}
	if !strings.Contains(seen, `<osm><node`) || !strings.Contains(seen, `changeset="5"`) {
		t.Errorf("body: %s", seen)
	}
}

func TestDeleteNode(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/node/9" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		io.WriteString(w, "4")
	}))
	v, err := c.DeleteNode(context.Background(), 5, &osm.Node{ID: 9, Version: 3})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if v != 4 {
		t.Errorf("got version %d", v)
	}
}

func TestVersionConflictMapsToErrConflict(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Version mismatch: Provided 2, server had: 3", http.StatusConflict)
	}))
	_, err := c.ModifyNode(context.Background(), 5, &osm.Node{ID: 1, Version: 2})
	if !errors.Is(err, ErrConflict) {
		t.Errorf("got %v, want errors.Is(err, ErrConflict)", err)
	}
}
