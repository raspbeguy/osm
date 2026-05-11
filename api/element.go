package api

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/paulmach/osm"
)

func (c *Client) NodeHistory(ctx context.Context, id osm.NodeID) ([]*osm.Node, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/node/%d/history", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Nodes) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Nodes, nil
}

func (c *Client) WayHistory(ctx context.Context, id osm.WayID) ([]*osm.Way, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/way/%d/history", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Ways) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Ways, nil
}

func (c *Client) RelationHistory(ctx context.Context, id osm.RelationID) ([]*osm.Relation, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/relation/%d/history", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Relations) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Relations, nil
}

func (c *Client) GetNode(ctx context.Context, id osm.NodeID) (*osm.Node, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/node/%d", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Nodes) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Nodes[0], nil
}

func (c *Client) GetWay(ctx context.Context, id osm.WayID) (*osm.Way, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/way/%d", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Ways) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Ways[0], nil
}

func (c *Client) GetRelation(ctx context.Context, id osm.RelationID) (*osm.Relation, error) {
	var wrap osm.OSM
	if err := c.getXML(ctx, fmt.Sprintf("/relation/%d", id), &wrap); err != nil {
		return nil, err
	}
	if len(wrap.Relations) == 0 {
		return nil, ErrNotFound
	}
	return wrap.Relations[0], nil
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
