package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

type Note struct {
	ID        int64
	Status    string
	Lat, Lon  float64
	CreatedAt string
	ClosedAt  string
	Comments  []NoteComment
}

type NoteComment struct {
	Date   string
	UID    int64
	User   string
	Action string
	Text   string
}

type NotesQuery struct {
	BBox   [4]float64 // left, bottom, right, top in WGS84 degrees
	Limit  int        // 1..10000; zero leaves the server default
	Closed int        // days since close to include; -1 for all, 0 for only open
}

type noteFeature struct {
	Geometry struct {
		Coordinates [2]float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		ID          int64  `json:"id"`
		Status      string `json:"status"`
		DateCreated string `json:"date_created"`
		DateClosed  string `json:"date_closed,omitempty"`
		Comments    []struct {
			Date   string `json:"date"`
			UID    int64  `json:"uid"`
			User   string `json:"user"`
			Action string `json:"action"`
			Text   string `json:"text"`
		} `json:"comments"`
	} `json:"properties"`
}

func (f noteFeature) toNote() *Note {
	n := &Note{
		ID:        f.Properties.ID,
		Status:    f.Properties.Status,
		Lat:       f.Geometry.Coordinates[1],
		Lon:       f.Geometry.Coordinates[0],
		CreatedAt: f.Properties.DateCreated,
		ClosedAt:  f.Properties.DateClosed,
	}
	for _, c := range f.Properties.Comments {
		n.Comments = append(n.Comments, NoteComment{
			Date: c.Date, UID: c.UID, User: c.User, Action: c.Action, Text: c.Text,
		})
	}
	return n
}

func (c *Client) GetNote(ctx context.Context, id int64) (*Note, error) {
	var f noteFeature
	if err := c.getJSON(ctx, fmt.Sprintf("/notes/%d.json", id), &f); err != nil {
		return nil, err
	}
	return f.toNote(), nil
}

func (c *Client) QueryNotes(ctx context.Context, q NotesQuery) ([]*Note, error) {
	p := url.Values{}
	p.Set("bbox", fmt.Sprintf("%g,%g,%g,%g", q.BBox[0], q.BBox[1], q.BBox[2], q.BBox[3]))
	if q.Limit > 0 {
		p.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Closed != 0 {
		p.Set("closed", strconv.Itoa(q.Closed))
	}
	var wrap struct {
		Features []noteFeature `json:"features"`
	}
	if err := c.getJSON(ctx, "/notes.json?"+p.Encode(), &wrap); err != nil {
		return nil, err
	}
	out := make([]*Note, len(wrap.Features))
	for i, f := range wrap.Features {
		out[i] = f.toNote()
	}
	return out, nil
}

func (c *Client) CreateNote(ctx context.Context, lat, lon float64, text string) (*Note, error) {
	if text == "" {
		return nil, errors.New("note text required")
	}
	p := url.Values{}
	p.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	p.Set("lon", strconv.FormatFloat(lon, 'f', -1, 64))
	p.Set("text", text)
	return c.postNote(ctx, "/notes.json?"+p.Encode())
}

func (c *Client) CommentNote(ctx context.Context, id int64, text string) (*Note, error) {
	return c.noteAction(ctx, id, "comment", text)
}

func (c *Client) CloseNote(ctx context.Context, id int64, text string) (*Note, error) {
	return c.noteAction(ctx, id, "close", text)
}

func (c *Client) ReopenNote(ctx context.Context, id int64, text string) (*Note, error) {
	return c.noteAction(ctx, id, "reopen", text)
}

func (c *Client) noteAction(ctx context.Context, id int64, action, text string) (*Note, error) {
	path := fmt.Sprintf("/notes/%d/%s.json", id, action)
	if text != "" {
		path += "?" + url.Values{"text": {text}}.Encode()
	}
	return c.postNote(ctx, path)
}

func (c *Client) postNote(ctx context.Context, path string) (*Note, error) {
	body, err := c.sendBody(ctx, "POST", path, nil, "")
	if err != nil {
		return nil, err
	}
	var f noteFeature
	if err := json.Unmarshal([]byte(body), &f); err != nil {
		return nil, fmt.Errorf("decode note: %w", err)
	}
	return f.toNote(), nil
}
