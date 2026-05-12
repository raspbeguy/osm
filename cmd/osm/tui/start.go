package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

// startAt parses positional args from `osm tui` and returns the tea.Cmd that
// navigates to the requested screen on program start. nil cmd means stay on
// the main menu.
func startAt(args []string) (tea.Cmd, error) {
	if len(args) == 0 {
		return nil, nil
	}
	target := args[0]
	rest := args[1:]
	nav := func(s screen, refresh bool) tea.Cmd {
		return func() tea.Msg { return navigateMsg{to: s, refresh: refresh} }
	}
	switch target {
	case "menu":
		return nil, nil
	case "profile":
		return nav(screenProfile, false), nil
	case "inbox":
		return nav(screenInbox, true), nil
	case "outbox":
		return nav(screenOutbox, true), nil
	case "changesets":
		return nav(screenChangesets, true), nil
	case "changeset":
		if len(rest) != 1 {
			return nil, fmt.Errorf("usage: osm tui changeset <id>")
		}
		id, err := strconv.ParseInt(rest[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid changeset id %q: %w", rest[0], err)
		}
		return func() tea.Msg { return navigateMsg{to: screenChangesetView, itemID: id} }, nil
	case "notes":
		return nav(screenNotes, false), nil
	case "doctor":
		return nav(screenDoctor, false), nil
	case "history":
		if len(rest) != 2 {
			return nil, fmt.Errorf("usage: osm tui history <kind> <id>")
		}
		kind := rest[0]
		if kind != "node" && kind != "way" && kind != "relation" {
			return nil, fmt.Errorf("kind must be node, way, or relation; got %q", kind)
		}
		id, err := strconv.ParseInt(rest[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id %q: %w", rest[1], err)
		}
		return func() tea.Msg { return navigateMsg{to: screenHistory, kind: kind, itemID: id} }, nil
	case "traces":
		return nav(screenTraces, true), nil
	case "compose", "new":
		return nav(screenComposeChangeset, false), nil
	}
	return nil, fmt.Errorf("unknown target %q", target)
}
