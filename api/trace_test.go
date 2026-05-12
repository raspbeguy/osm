package api

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleTracesXML = `<osm><gpx_file id="42" name="ride.gpx" lat="48.5" lon="6.9" user="raspbeguy" uid="3398417" visibility="public" pending="false" timestamp="2024-01-01T00:00:00Z"><description>a ride</description><tag>bike</tag><tag>forest</tag></gpx_file></osm>`

func TestListTraces(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/gpx_files" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, sampleTracesXML)
	}))
	traces, err := c.ListTraces(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(traces) != 1 || traces[0].ID != 42 || traces[0].Description != "a ride" || len(traces[0].Tags) != 2 {
		t.Errorf("got %+v", traces)
	}
}

func TestGetTrace(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gpx/42/details" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, sampleTracesXML)
	}))
	tr, err := c.GetTrace(context.Background(), 42)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if tr.ID != 42 || tr.Visibility != "public" {
		t.Errorf("got %+v", tr)
	}
}

func TestGetTraceDataRedirect(t *testing.T) {
	var s3Hit bool
	s3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s3Hit = true
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("auth header leaked to redirect target: %q", got)
		}
		io.WriteString(w, "<gpx>blob</gpx>")
	}))
	t.Cleanup(s3.Close)
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gpx/42/data" {
			t.Errorf("path=%s", r.URL.Path)
		}
		w.Header().Set("Location", s3.URL)
		w.WriteHeader(http.StatusFound)
	}))
	data, err := c.GetTraceData(context.Background(), 42)
	if err != nil {
		t.Fatalf("data: %v", err)
	}
	if !s3Hit {
		t.Error("redirect target not hit")
	}
	if data != "<gpx>blob</gpx>" {
		t.Errorf("data=%q", data)
	}
}

func TestUploadTrace(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/gpx/create" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("content-type: %v", err)
		}
		mr := multipart.NewReader(r.Body, params["boundary"])
		fields := map[string]string{}
		var fileSize int
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("part: %v", err)
			}
			b, _ := io.ReadAll(p)
			if p.FileName() != "" {
				fileSize = len(b)
				continue
			}
			fields[p.FormName()] = string(b)
		}
		if fields["description"] != "test ride" || fields["visibility"] != "public" {
			t.Errorf("fields: %v", fields)
		}
		if fields["tags"] != "bike,forest" {
			t.Errorf("tags=%q", fields["tags"])
		}
		if fileSize == 0 {
			t.Error("no file uploaded")
		}
		io.WriteString(w, "777")
	}))
	id, err := c.UploadTrace(context.Background(), TraceUpload{
		Filename:    "ride.gpx",
		GPX:         []byte("<gpx></gpx>"),
		Description: "test ride",
		Tags:        []string{"bike", "forest"},
		Visibility:  "public",
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if id != 777 {
		t.Errorf("got id %d", id)
	}
}

func TestDeleteTrace(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/gpx/42" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
	}))
	if err := c.DeleteTrace(context.Background(), 42); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateTrace(t *testing.T) {
	var seen string
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/gpx/42" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
	}))
	if err := c.UpdateTrace(context.Background(), 42, TraceUpdate{
		Description: "new desc",
		Tags:        []string{"a", "b"},
		Visibility:  "private",
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(seen, "new desc") || !strings.Contains(seen, `visibility="private"`) {
		t.Errorf("body: %s", seen)
	}
}
