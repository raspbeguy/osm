package api

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestListInbox(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/messages/inbox.json" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, `{"messages":[{"id":1,"from_display_name":"a","title":"hi","message_read":false}]}`)
	}))
	msgs, err := c.ListInbox(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != 1 || msgs[0].FromUser != "a" || msgs[0].Read {
		t.Errorf("got %+v", msgs)
	}
}

func TestGetMessage(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user/messages/7.json" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, `{"message":{"id":7,"title":"t","body":"hello","body_format":"markdown","message_read":true}}`)
	}))
	m, err := c.GetMessage(context.Background(), 7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if m.ID != 7 || m.Body != "hello" || !m.Read {
		t.Errorf("got %+v", m)
	}
}

func TestMarkRead(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/user/messages/3" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type=%s", ct)
		}
		_ = r.ParseForm()
		if r.PostForm.Get("read_status") != "read" {
			t.Errorf("read_status=%s", r.PostForm.Get("read_status"))
		}
	}))
	if err := c.MarkRead(context.Background(), 3, true); err != nil {
		t.Fatalf("mark: %v", err)
	}
}

func TestDeleteMessage(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/user/messages/9" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
	}))
	if err := c.DeleteMessage(context.Background(), 9); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
