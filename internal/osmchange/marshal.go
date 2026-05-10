// Package osmchange marshals paulmach/osm Change values into the osmChange
// XML format the OSM API expects on upload.
package osmchange

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"github.com/paulmach/osm"
)

// Marshal returns an osmChange XML document. Every element gets the given
// changeset ID stamped onto it, since the API rejects uploads with mismatched
// or missing changeset attributes.
func Marshal(changesetID osm.ChangesetID, ch *osm.Change) ([]byte, error) {
	if ch == nil {
		return nil, fmt.Errorf("nil change")
	}
	stamp(changesetID, ch.Create)
	stamp(changesetID, ch.Modify)
	stamp(changesetID, ch.Delete)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	if err := enc.Encode(ch); err != nil {
		return nil, fmt.Errorf("encode osmChange: %w", err)
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func stamp(id osm.ChangesetID, o *osm.OSM) {
	if o == nil {
		return
	}
	for _, n := range o.Nodes {
		n.ChangesetID = id
	}
	for _, w := range o.Ways {
		w.ChangesetID = id
	}
	for _, r := range o.Relations {
		r.ChangesetID = id
	}
}
