package api

import (
	"context"
	"fmt"
)

// DownloadMap returns every visible feature inside bbox as OSM XML.
// bbox is [west, south, east, north] in WGS84 degrees.
func (c *Client) DownloadMap(ctx context.Context, bbox [4]float64) (string, error) {
	return c.getRaw(ctx, fmt.Sprintf("/map?bbox=%g,%g,%g,%g", bbox[0], bbox[1], bbox[2], bbox[3]), "application/xml")
}

type Capabilities struct {
	Version              string
	MinAPIVersion        string
	MaxAPIVersion        string
	AreaMax              float64
	NoteAreaMax          float64
	TracepointsPerPage   int
	WayNodesMax          int
	RelationMembersMax   int
	ChangesetMaxElements int
	ChangesetMaxQuery    int
	NotesMaxQuery        int
	TimeoutSeconds       int
	DatabaseStatus       string
	APIStatus            string
	GPXStatus            string
}

// Capabilities returns the server's advertised limits and status.
func (c *Client) Capabilities(ctx context.Context) (*Capabilities, error) {
	var wrap struct {
		Version string `json:"version"`
		API     struct {
			Version struct {
				Minimum string `json:"minimum"`
				Maximum string `json:"maximum"`
			} `json:"version"`
			Area struct {
				Maximum float64 `json:"maximum"`
			} `json:"area"`
			NoteArea struct {
				Maximum float64 `json:"maximum"`
			} `json:"note_area"`
			Tracepoints struct {
				PerPage int `json:"per_page"`
			} `json:"tracepoints"`
			WayNodes struct {
				Maximum int `json:"maximum"`
			} `json:"waynodes"`
			RelationMembers struct {
				Maximum int `json:"maximum"`
			} `json:"relationmembers"`
			Changesets struct {
				MaximumElements   int `json:"maximum_elements"`
				MaximumQueryLimit int `json:"maximum_query_limit"`
			} `json:"changesets"`
			Notes struct {
				MaximumQueryLimit int `json:"maximum_query_limit"`
			} `json:"notes"`
			Timeout struct {
				Seconds int `json:"seconds"`
			} `json:"timeout"`
			Status struct {
				Database string `json:"database"`
				API      string `json:"api"`
				GPX      string `json:"gpx"`
			} `json:"status"`
		} `json:"api"`
	}
	if err := c.getJSON(ctx, "/capabilities.json", &wrap); err != nil {
		return nil, err
	}
	return &Capabilities{
		Version:              wrap.Version,
		MinAPIVersion:        wrap.API.Version.Minimum,
		MaxAPIVersion:        wrap.API.Version.Maximum,
		AreaMax:              wrap.API.Area.Maximum,
		NoteAreaMax:          wrap.API.NoteArea.Maximum,
		TracepointsPerPage:   wrap.API.Tracepoints.PerPage,
		WayNodesMax:          wrap.API.WayNodes.Maximum,
		RelationMembersMax:   wrap.API.RelationMembers.Maximum,
		ChangesetMaxElements: wrap.API.Changesets.MaximumElements,
		ChangesetMaxQuery:    wrap.API.Changesets.MaximumQueryLimit,
		NotesMaxQuery:        wrap.API.Notes.MaximumQueryLimit,
		TimeoutSeconds:       wrap.API.Timeout.Seconds,
		DatabaseStatus:       wrap.API.Status.Database,
		APIStatus:            wrap.API.Status.API,
		GPXStatus:            wrap.API.Status.GPX,
	}, nil
}

// Permissions returns the OAuth scope names the current token carries.
func (c *Client) Permissions(ctx context.Context) ([]string, error) {
	var wrap struct {
		Permissions []string `json:"permissions"`
	}
	if err := c.getJSON(ctx, "/permissions.json", &wrap); err != nil {
		return nil, err
	}
	return wrap.Permissions, nil
}
