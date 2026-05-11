package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

type Trace struct {
	ID          int64
	Name        string
	Description string
	Tags        []string
	User        string
	UserID      int64
	Visibility  string
	Pending     bool
	Timestamp   string
	Lat         float64
	Lon         float64
}

type TraceUpload struct {
	Filename    string
	GPX         []byte
	Description string
	Tags        []string
	Visibility  string // "public" | "trackable" | "identifiable" | "private". Empty -> "private".
}

type TraceUpdate struct {
	Description string
	Tags        []string
	Visibility  string
}

type traceXML struct {
	ID          int64    `xml:"id,attr"`
	Name        string   `xml:"name,attr"`
	User        string   `xml:"user,attr"`
	UID         int64    `xml:"uid,attr"`
	Visibility  string   `xml:"visibility,attr"`
	Pending     bool     `xml:"pending,attr"`
	Timestamp   string   `xml:"timestamp,attr"`
	Lat         float64  `xml:"lat,attr"`
	Lon         float64  `xml:"lon,attr"`
	Description string   `xml:"description"`
	Tags        []string `xml:"tag"`
}

func (t traceXML) toTrace() *Trace {
	return &Trace{
		ID: t.ID, Name: t.Name, User: t.User, UserID: t.UID,
		Visibility: t.Visibility, Pending: t.Pending,
		Timestamp: t.Timestamp, Lat: t.Lat, Lon: t.Lon,
		Description: t.Description, Tags: t.Tags,
	}
}

type traceWrap struct {
	XMLName xml.Name   `xml:"osm"`
	Traces  []traceXML `xml:"gpx_file"`
}

// ListTraces returns the authenticated user's uploaded GPS traces.
func (c *Client) ListTraces(ctx context.Context) ([]*Trace, error) {
	var wrap traceWrap
	if err := c.getXML(ctx, "/user/gpx_files", &wrap); err != nil {
		return nil, err
	}
	out := make([]*Trace, len(wrap.Traces))
	for i, t := range wrap.Traces {
		out[i] = t.toTrace()
	}
	return out, nil
}

func (c *Client) GetTrace(ctx context.Context, id int64) (*Trace, error) {
	var wrap traceWrap
	if err := c.getXML(ctx, fmt.Sprintf("/gpx/%d/details", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Traces) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Traces[0].toTrace(), nil
}

// GetTraceData returns the raw GPX file content. The OSM API often redirects
// to an S3 URL for the actual bytes, so we follow that redirect manually with
// an unauthenticated request to avoid S3 rejecting the OSM Bearer token.
func (c *Client) GetTraceData(ctx context.Context, id int64) (string, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/gpx/%d/data", id), nil)
	if err != nil {
		return "", err
	}
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	noFollow := &http.Client{
		Transport: c.HTTP.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := noFollow.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		b, err := io.ReadAll(resp.Body)
		return string(b), err
	case http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		loc := resp.Header.Get("Location")
		if loc == "" {
			return "", fmt.Errorf("redirect without location")
		}
		r2, err := http.NewRequestWithContext(ctx, http.MethodGet, loc, nil)
		if err != nil {
			return "", err
		}
		resp2, err := http.DefaultClient.Do(r2)
		if err != nil {
			return "", err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode >= 400 {
			body, _ := io.ReadAll(io.LimitReader(resp2.Body, 4096))
			return "", fmt.Errorf("fetch %s: %s: %s", loc, resp2.Status, string(body))
		}
		b, err := io.ReadAll(resp2.Body)
		return string(b), err
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", mapHTTPError(resp.StatusCode, resp.Status, string(body))
	}
}

func (c *Client) DeleteTrace(ctx context.Context, id int64) error {
	_, err := c.sendBody(ctx, "DELETE", fmt.Sprintf("/gpx/%d", id), nil, "")
	return err
}

// UploadTrace posts a new GPX file. Returns the assigned trace ID.
func (c *Client) UploadTrace(ctx context.Context, t TraceUpload) (int64, error) {
	if t.Description == "" {
		return 0, errors.New("description required")
	}
	if len(t.GPX) == 0 {
		return 0, errors.New("empty gpx")
	}
	if t.Visibility == "" {
		t.Visibility = "private"
	}
	filename := t.Filename
	if filename == "" {
		filename = "trace.gpx"
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("description", t.Description); err != nil {
		return 0, err
	}
	if err := mw.WriteField("tags", strings.Join(t.Tags, ",")); err != nil {
		return 0, err
	}
	if err := mw.WriteField("visibility", t.Visibility); err != nil {
		return 0, err
	}
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		return 0, err
	}
	if _, err := fw.Write(t.GPX); err != nil {
		return 0, err
	}
	if err := mw.Close(); err != nil {
		return 0, err
	}

	out, err := c.sendBody(ctx, "POST", "/gpx/create", body.Bytes(), mw.FormDataContentType())
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse trace id %q: %w", out, err)
	}
	return id, nil
}

// UpdateTrace replaces the metadata of an existing trace.
func (c *Client) UpdateTrace(ctx context.Context, id int64, u TraceUpdate) error {
	type tagX struct {
		XMLName xml.Name `xml:"tag"`
		Value   string   `xml:",chardata"`
	}
	type gpxFileX struct {
		XMLName     xml.Name `xml:"gpx_file"`
		ID          int64    `xml:"id,attr"`
		Visibility  string   `xml:"visibility,attr,omitempty"`
		Description string   `xml:"description"`
		Tags        []tagX   `xml:"tag,omitempty"`
	}
	type wrap struct {
		XMLName xml.Name `xml:"osm"`
		File    gpxFileX `xml:"gpx_file"`
	}
	w := wrap{File: gpxFileX{ID: id, Visibility: u.Visibility, Description: u.Description}}
	for _, t := range u.Tags {
		w.File.Tags = append(w.File.Tags, tagX{Value: t})
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(w); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	_, err := c.sendBody(ctx, "PUT", fmt.Sprintf("/gpx/%d", id), buf.Bytes(), "application/xml")
	return err
}
