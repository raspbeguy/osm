package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/paulmach/osm"
	"github.com/spf13/cobra"

	osmapi "github.com/raspbeguy/osm/api"
)

var editCmd = &cobra.Command{Use: "edit", Short: "modify osm elements (opens and closes a one-shot changeset)"}

var editComment string

var editTagCmd = &cobra.Command{
	Use:   "tag <node|way|relation> <id> <key=value> [key=value ...]",
	Short: "set or update tags on an existing element (empty value deletes a key)",
	Args:  cobra.MinimumNArgs(3),
	RunE:  runEditTag,
}

var editDeleteCmd = &cobra.Command{
	Use:   "delete <node|way|relation> <id>",
	Short: "delete an existing element",
	Args:  cobra.ExactArgs(2),
	RunE:  runEditDelete,
}

func runEditTag(cmd *cobra.Command, args []string) error {
	kind, idStr := args[0], args[1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return err
	}
	patch, err := parseKVTags(args[2:])
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd.Context())
	if err != nil {
		return err
	}
	return withChangeset(cmd.Context(), c, editComment, func(csID osm.ChangesetID) error {
		switch kind {
		case "node":
			n, err := c.GetNode(cmd.Context(), osm.NodeID(id))
			if err != nil {
				return err
			}
			mergeTags(&n.Tags, patch)
			_, err = c.ModifyNode(cmd.Context(), csID, n)
			return err
		case "way":
			w, err := c.GetWay(cmd.Context(), osm.WayID(id))
			if err != nil {
				return err
			}
			mergeTags(&w.Tags, patch)
			_, err = c.ModifyWay(cmd.Context(), csID, w)
			return err
		case "relation":
			r, err := c.GetRelation(cmd.Context(), osm.RelationID(id))
			if err != nil {
				return err
			}
			mergeTags(&r.Tags, patch)
			_, err = c.ModifyRelation(cmd.Context(), csID, r)
			return err
		default:
			return errUnknownKind(kind)
		}
	})
}

func errUnknownKind(kind string) error {
	return fmt.Errorf("unknown element kind %q (want node, way, or relation)", kind)
}

func runEditDelete(cmd *cobra.Command, args []string) error {
	kind, idStr := args[0], args[1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return err
	}
	c, err := newAPIClient(cmd.Context())
	if err != nil {
		return err
	}
	return withChangeset(cmd.Context(), c, editComment, func(csID osm.ChangesetID) error {
		switch kind {
		case "node":
			n, err := c.GetNode(cmd.Context(), osm.NodeID(id))
			if err != nil {
				return err
			}
			_, err = c.DeleteNode(cmd.Context(), csID, n)
			return err
		case "way":
			w, err := c.GetWay(cmd.Context(), osm.WayID(id))
			if err != nil {
				return err
			}
			_, err = c.DeleteWay(cmd.Context(), csID, w)
			return err
		case "relation":
			r, err := c.GetRelation(cmd.Context(), osm.RelationID(id))
			if err != nil {
				return err
			}
			_, err = c.DeleteRelation(cmd.Context(), csID, r)
			return err
		default:
			return errUnknownKind(kind)
		}
	})
}

func parseKVTags(args []string) (osm.Tags, error) {
	out := osm.Tags{}
	for _, a := range args {
		i := strings.IndexByte(a, '=')
		if i < 0 {
			return nil, fmt.Errorf("expected key=value, got %q", a)
		}
		key := a[:i]
		if key == "" {
			return nil, fmt.Errorf("empty tag key in %q", a)
		}
		out = append(out, osm.Tag{Key: key, Value: a[i+1:]})
	}
	return out, nil
}

// mergeTags applies patch onto dst. An empty value removes the key.
func mergeTags(dst *osm.Tags, patch osm.Tags) {
	for _, t := range patch {
		idx := -1
		for i, e := range *dst {
			if e.Key == t.Key {
				idx = i
				break
			}
		}
		if t.Value == "" {
			if idx >= 0 {
				*dst = append((*dst)[:idx], (*dst)[idx+1:]...)
			}
			continue
		}
		if idx >= 0 {
			(*dst)[idx].Value = t.Value
		} else {
			*dst = append(*dst, t)
		}
	}
}

func withChangeset(ctx context.Context, c *osmapi.Client, comment string, fn func(osm.ChangesetID) error) error {
	tags := osm.Tags{{Key: "created_by", Value: "osm-go"}}
	if comment != "" {
		tags = append(tags, osm.Tag{Key: "comment", Value: comment})
	}
	_, err := c.WithChangeset(ctx, tags, fn)
	return err
}

func init() {
	editCmd.PersistentFlags().StringVar(&editComment, "comment", "", "changeset comment")
	editCmd.AddCommand(editTagCmd, editDeleteCmd)
	rootCmd.AddCommand(editCmd)
}
