package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/paulmach/osm"
)

// getOne fetches the OSM XML at path and returns the first element from the
// slice produced by pick. ErrNotFound on empty.
func getOne[T any, S ~[]*T](c *Client, ctx context.Context, path string, pick func(*osm.OSM) S) (*T, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, path, &wrap); err != nil {
		return nil, err
	}
	items := pick(&wrap)
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items[0], nil
}

// getAll fetches the OSM XML at path and returns the non-empty slice produced
// by pick. ErrNotFound on empty (used by element history endpoints).
func getAll[T any, S ~[]*T](c *Client, ctx context.Context, path string, pick func(*osm.OSM) S) ([]*T, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, path, &wrap); err != nil {
		return nil, err
	}
	items := pick(&wrap)
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

// GetNodes fetches multiple nodes in one request. Order matches the response,
// not the input.
func (c *Client) GetNodes(ctx context.Context, ids []osm.NodeID) ([]*osm.Node, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one id required")
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(int64(id), 10)
	}
	var wrap osm.OSM
	if err := c.getXML(ctx, "/nodes?nodes="+strings.Join(parts, ","), &wrap); err != nil {
		return nil, err
	}
	return wrap.Nodes, nil
}

func (c *Client) GetWays(ctx context.Context, ids []osm.WayID) ([]*osm.Way, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one id required")
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(int64(id), 10)
	}
	var wrap osm.OSM
	if err := c.getXML(ctx, "/ways?ways="+strings.Join(parts, ","), &wrap); err != nil {
		return nil, err
	}
	return wrap.Ways, nil
}

func (c *Client) GetRelations(ctx context.Context, ids []osm.RelationID) ([]*osm.Relation, error) {
	if len(ids) == 0 {
		return nil, errors.New("at least one id required")
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(int64(id), 10)
	}
	var wrap osm.OSM
	if err := c.getXML(ctx, "/relations?relations="+strings.Join(parts, ","), &wrap); err != nil {
		return nil, err
	}
	return wrap.Relations, nil
}

func (c *Client) GetNodeVersion(ctx context.Context, id osm.NodeID, version int) (*osm.Node, error) {
	return getOne(c, ctx, fmt.Sprintf("/node/%d/%d", id, version), func(o *osm.OSM) osm.Nodes { return o.Nodes })
}

func (c *Client) GetWayVersion(ctx context.Context, id osm.WayID, version int) (*osm.Way, error) {
	return getOne(c, ctx, fmt.Sprintf("/way/%d/%d", id, version), func(o *osm.OSM) osm.Ways { return o.Ways })
}

func (c *Client) GetRelationVersion(ctx context.Context, id osm.RelationID, version int) (*osm.Relation, error) {
	return getOne(c, ctx, fmt.Sprintf("/relation/%d/%d", id, version), func(o *osm.OSM) osm.Relations { return o.Relations })
}

func (c *Client) NodeHistory(ctx context.Context, id osm.NodeID) ([]*osm.Node, error) {
	return getAll(c, ctx, fmt.Sprintf("/node/%d/history", id), func(o *osm.OSM) osm.Nodes { return o.Nodes })
}

func (c *Client) WayHistory(ctx context.Context, id osm.WayID) ([]*osm.Way, error) {
	return getAll(c, ctx, fmt.Sprintf("/way/%d/history", id), func(o *osm.OSM) osm.Ways { return o.Ways })
}

func (c *Client) RelationHistory(ctx context.Context, id osm.RelationID) ([]*osm.Relation, error) {
	return getAll(c, ctx, fmt.Sprintf("/relation/%d/history", id), func(o *osm.OSM) osm.Relations { return o.Relations })
}

func (c *Client) GetNode(ctx context.Context, id osm.NodeID) (*osm.Node, error) {
	return getOne(c, ctx, fmt.Sprintf("/node/%d", id), func(o *osm.OSM) osm.Nodes { return o.Nodes })
}

func (c *Client) GetWay(ctx context.Context, id osm.WayID) (*osm.Way, error) {
	return getOne(c, ctx, fmt.Sprintf("/way/%d", id), func(o *osm.OSM) osm.Ways { return o.Ways })
}

func (c *Client) GetRelation(ctx context.Context, id osm.RelationID) (*osm.Relation, error) {
	return getOne(c, ctx, fmt.Sprintf("/relation/%d", id), func(o *osm.OSM) osm.Relations { return o.Relations })
}

func (c *Client) ModifyNode(ctx context.Context, csID osm.ChangesetID, n *osm.Node) (int, error) {
	n.ChangesetID = csID
	return c.putElement(ctx, "node", int64(n.ID), n)
}

func (c *Client) ModifyWay(ctx context.Context, csID osm.ChangesetID, w *osm.Way) (int, error) {
	w.ChangesetID = csID
	return c.putElement(ctx, "way", int64(w.ID), w)
}

func (c *Client) ModifyRelation(ctx context.Context, csID osm.ChangesetID, r *osm.Relation) (int, error) {
	r.ChangesetID = csID
	return c.putElement(ctx, "relation", int64(r.ID), r)
}

func (c *Client) DeleteNode(ctx context.Context, csID osm.ChangesetID, n *osm.Node) (int, error) {
	n.ChangesetID = csID
	return c.deleteElement(ctx, "node", int64(n.ID), n)
}

func (c *Client) DeleteWay(ctx context.Context, csID osm.ChangesetID, w *osm.Way) (int, error) {
	w.ChangesetID = csID
	return c.deleteElement(ctx, "way", int64(w.ID), w)
}

func (c *Client) DeleteRelation(ctx context.Context, csID osm.ChangesetID, r *osm.Relation) (int, error) {
	r.ChangesetID = csID
	return c.deleteElement(ctx, "relation", int64(r.ID), r)
}

func (c *Client) putElement(ctx context.Context, kind string, id int64, elem any) (int, error) {
	body, err := wrapElement(elem)
	if err != nil {
		return 0, err
	}
	out, err := c.sendBody(ctx, "PUT", fmt.Sprintf("/%s/%d", kind, id), body, "application/xml")
	if err != nil {
		return 0, err
	}
	return parseVersion(out)
}

func (c *Client) deleteElement(ctx context.Context, kind string, id int64, elem any) (int, error) {
	body, err := wrapElement(elem)
	if err != nil {
		return 0, err
	}
	out, err := c.sendBody(ctx, "DELETE", fmt.Sprintf("/%s/%d", kind, id), body, "application/xml")
	if err != nil {
		return 0, err
	}
	return parseVersion(out)
}

func wrapElement(elem any) ([]byte, error) {
	inner, err := xml.Marshal(elem)
	if err != nil {
		return nil, fmt.Errorf("encode element: %w", err)
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString("<osm>")
	buf.Write(inner)
	buf.WriteString("</osm>")
	return buf.Bytes(), nil
}

func parseVersion(s string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, fmt.Errorf("parse version %q: %w", s, err)
	}
	return v, nil
}
