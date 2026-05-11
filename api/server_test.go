package api

import (
	"context"
	"io"
	"net/http"
	"testing"
)

func TestCapabilities(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/capabilities.json" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, `{
            "version":"0.6","api":{
            "version":{"minimum":"0.6","maximum":"0.6"},
            "area":{"maximum":0.25},"note_area":{"maximum":25},
            "tracepoints":{"per_page":5000},
            "waynodes":{"maximum":2000},
            "relationmembers":{"maximum":32000},
            "changesets":{"maximum_elements":10000,"maximum_query_limit":100},
            "notes":{"maximum_query_limit":10000},
            "timeout":{"seconds":300},
            "status":{"database":"online","api":"online","gpx":"online"}
        }}`)
	}))
	caps, err := c.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("caps: %v", err)
	}
	if caps.ChangesetMaxElements != 10000 || caps.AreaMax != 0.25 || caps.APIStatus != "online" {
		t.Errorf("got %+v", caps)
	}
}

func TestPermissions(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/permissions.json" {
			t.Errorf("path=%s", r.URL.Path)
		}
		io.WriteString(w, `{"permissions":["allow_read_prefs","allow_write_api"]}`)
	}))
	perms, err := c.Permissions(context.Background())
	if err != nil {
		t.Fatalf("perms: %v", err)
	}
	if len(perms) != 2 || perms[0] != "allow_read_prefs" {
		t.Errorf("got %v", perms)
	}
}
