package api

import (
	"context"
	"io"
	"net/http"
	"testing"
)

const sampleNoteJSON = `{
  "type":"Feature",
  "geometry":{"type":"Point","coordinates":[6.946,48.582]},
  "properties":{
    "id":99,"status":"open","date_created":"2024-01-01 00:00:00 UTC",
    "comments":[{"date":"2024-01-01 00:00:00 UTC","uid":1,"user":"a","action":"opened","text":"hello"}]
  }
}`

func TestGetNote(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/notes/99.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		io.WriteString(w, sampleNoteJSON)
	}))
	n, err := c.GetNote(context.Background(), 99)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if n.ID != 99 || n.Status != "open" || n.Lat != 48.582 || n.Lon != 6.946 || len(n.Comments) != 1 {
		t.Errorf("got %+v", n)
	}
}

func TestCreateNote(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/notes.json" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("lat") != "1" || q.Get("lon") != "2" || q.Get("text") != "hi" {
			t.Errorf("params: %s", r.URL.RawQuery)
		}
		io.WriteString(w, sampleNoteJSON)
	}))
	n, err := c.CreateNote(context.Background(), 1, 2, "hi")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if n.ID != 99 {
		t.Errorf("got id %d", n.ID)
	}
}

func TestNoteActions(t *testing.T) {
	var seen string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.URL.Path
		io.WriteString(w, sampleNoteJSON)
	}))
	if _, err := c.CommentNote(context.Background(), 99, "x"); err != nil {
		t.Fatal(err)
	}
	if seen != "/notes/99/comment.json" {
		t.Errorf("got %s", seen)
	}
	if _, err := c.CloseNote(context.Background(), 99, ""); err != nil {
		t.Fatal(err)
	}
	if seen != "/notes/99/close.json" {
		t.Errorf("got %s", seen)
	}
	if _, err := c.ReopenNote(context.Background(), 99, ""); err != nil {
		t.Fatal(err)
	}
	if seen != "/notes/99/reopen.json" {
		t.Errorf("got %s", seen)
	}
}

func TestQueryNotes(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("bbox") != "6,48,7,49" {
			t.Errorf("bbox=%q", r.URL.Query().Get("bbox"))
		}
		io.WriteString(w, `{"features":[`+sampleNoteJSON+`]}`)
	}))
	notes, err := c.QueryNotes(context.Background(), NotesQuery{BBox: [4]float64{6, 48, 7, 49}, Limit: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(notes) != 1 || notes[0].ID != 99 {
		t.Errorf("got %+v", notes)
	}
}
