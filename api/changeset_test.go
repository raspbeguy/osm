package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/paulmach/osm"
)

func TestOpenChangeset(t *testing.T) {
	var seenBody string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/changeset/create" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		seenBody = string(b)
		io.WriteString(w, "12345")
	}))
	id, err := c.OpenChangeset(context.Background(), osm.Tags{
		{Key: "comment", Value: "test"},
		{Key: "created_by", Value: "go-osm"},
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if id != 12345 {
		t.Errorf("got id %d", id)
	}
	if !strings.Contains(seenBody, `k="comment"`) || !strings.Contains(seenBody, `<changeset>`) {
		t.Errorf("body missing tags: %s", seenBody)
	}
}

func TestUploadChange(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/changeset/77/upload" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `changeset="77"`) {
			t.Errorf("body missing changeset stamp: %s", string(b))
		}
		io.WriteString(w, `<diffResult/>`)
	}))
	ch := &osm.Change{
		Modify: &osm.OSM{
			Nodes: osm.Nodes{{ID: 1, Version: 2, Lat: 0, Lon: 0}},
		},
	}
	out, err := c.UploadChange(context.Background(), 77, ch)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if !strings.Contains(out, "diffResult") {
		t.Errorf("unexpected body: %s", out)
	}
}

func TestCloseAndComment(t *testing.T) {
	var calls []string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
	}))
	if err := c.CloseChangeset(context.Background(), 5); err != nil {
		t.Fatal(err)
	}
	if err := c.CommentChangeset(context.Background(), 5, "nice work"); err != nil {
		t.Fatal(err)
	}
	want := []string{"PUT /changeset/5/close", "POST /changeset/5/comment"}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("got %v", calls)
	}
}

func TestWithChangesetClosesOnError(t *testing.T) {
	var opened, closed bool
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/changeset/create":
			opened = true
			io.WriteString(w, "42")
		case "/changeset/42/close":
			closed = true
		}
	}))
	wantErr := io.ErrUnexpectedEOF
	id, err := c.WithChangeset(context.Background(), nil, func(id osm.ChangesetID) error {
		if id != 42 {
			t.Errorf("inner id = %d", id)
		}
		return wantErr
	})
	if err != wantErr {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
	if id != 42 || !opened || !closed {
		t.Errorf("opened=%v closed=%v id=%d", opened, closed, id)
	}
}
