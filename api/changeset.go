package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/paulmach/osm"
	"github.com/raspbeguy/osm/internal/osmchange"
)

// OpenChangeset creates a changeset with the given tags and returns its ID.
// Conventional tags: "comment", "source", "created_by".
func (c *Client) OpenChangeset(ctx context.Context, tags osm.Tags) (osm.ChangesetID, error) {
	body, err := buildChangesetCreate(tags)
	if err != nil {
		return 0, err
	}
	out, err := c.sendBody(ctx, "PUT", "/changeset/create", body, "application/xml")
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(strings.TrimSpace(out), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse changeset id %q: %w", out, err)
	}
	return osm.ChangesetID(id), nil
}

func (c *Client) CloseChangeset(ctx context.Context, id osm.ChangesetID) error {
	_, err := c.sendBody(ctx, "PUT", fmt.Sprintf("/changeset/%d/close", id), nil, "")
	return err
}

// UploadChange uploads the given Change against an open changeset. The raw
// diffResult XML is returned for callers that want to map new IDs back.
func (c *Client) UploadChange(ctx context.Context, id osm.ChangesetID, change *osm.Change) (string, error) {
	body, err := osmchange.Marshal(id, change)
	if err != nil {
		return "", err
	}
	return c.sendBody(ctx, "POST", fmt.Sprintf("/changeset/%d/upload", id), body, "application/xml")
}

func (c *Client) CommentChangeset(ctx context.Context, id osm.ChangesetID, text string) error {
	if text == "" {
		return errors.New("comment text required")
	}
	body := url.Values{"text": {text}}.Encode()
	_, err := c.sendBody(ctx, "POST", fmt.Sprintf("/changeset/%d/comment", id), []byte(body), "application/x-www-form-urlencoded")
	return err
}

// DownloadChangeset returns the raw osmChange XML that was uploaded in this changeset.
func (c *Client) DownloadChangeset(ctx context.Context, id osm.ChangesetID) (string, error) {
	return c.getRaw(ctx, fmt.Sprintf("/changeset/%d/download", id), "application/xml")
}

// GetChangeset returns the changeset including its discussion (comments).
func (c *Client) GetChangeset(ctx context.Context, id osm.ChangesetID) (*osm.Changeset, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/changeset/%d?include_discussion=true", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Changesets) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Changesets[0], nil
}

type ChangesetFilter struct {
	UserID      int64
	DisplayName string
	OnlyOpen    bool
	OnlyClosed  bool
	Limit       int
}

func (c *Client) ListChangesets(ctx context.Context, f ChangesetFilter) ([]*osm.Changeset, error) {
	if f.OnlyOpen && f.OnlyClosed {
		return nil, errors.New("ChangesetFilter.OnlyOpen and OnlyClosed are mutually exclusive")
	}
	q := url.Values{}
	switch {
	case f.UserID > 0:
		q.Set("user", strconv.FormatInt(f.UserID, 10))
	case f.DisplayName != "":
		q.Set("display_name", f.DisplayName)
	}
	if f.OnlyOpen {
		q.Set("open", "true")
	}
	if f.OnlyClosed {
		q.Set("closed", "true")
	}
	if f.Limit > 0 {
		q.Set("limit", strconv.Itoa(f.Limit))
	}
	path := "/changesets"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var wrap osm.OSM
	if err := c.getXML(ctx, path, &wrap); err != nil {
		return nil, err
	}
	return wrap.Changesets, nil
}

// WithChangeset opens a changeset, calls fn, and closes the changeset even if fn errors.
// Both errors are surfaced via errors.Join so a half-closed changeset is visible to the caller.
func (c *Client) WithChangeset(ctx context.Context, tags osm.Tags, fn func(id osm.ChangesetID) error) (osm.ChangesetID, error) {
	id, err := c.OpenChangeset(ctx, tags)
	if err != nil {
		return 0, err
	}
	fnErr := fn(id)
	closeErr := c.CloseChangeset(ctx, id)
	return id, errors.Join(fnErr, closeErr)
}

func buildChangesetCreate(tags osm.Tags) ([]byte, error) {
	type xtag struct {
		XMLName xml.Name `xml:"tag"`
		K       string   `xml:"k,attr"`
		V       string   `xml:"v,attr"`
	}
	type xcs struct {
		XMLName xml.Name `xml:"changeset"`
		Tags    []xtag   `xml:"tag"`
	}
	type xwrap struct {
		XMLName xml.Name `xml:"osm"`
		CS      xcs      `xml:"changeset"`
	}
	w := xwrap{}
	for _, t := range tags {
		w.CS.Tags = append(w.CS.Tags, xtag{K: t.Key, V: t.Value})
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(w); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
