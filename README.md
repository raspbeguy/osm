# osm

A command-line client and TUI for OpenStreetMap, plus the Go libraries
underneath. The libraries fill a gap in the existing ecosystem: `paulmach/osm`
covers the read side and file parsing, but there is no Go equivalent of
Python's `osmapi` for authenticated writes. This module adds OAuth2 PKCE, the
full API v0.6 client surface (changesets, elements, notes, messages, traces,
user preferences), and an interactive TUI on top.

## Install

```
go install github.com/raspbeguy/osm/cmd/osm@latest
```

Or build from a checkout:

```
git clone https://github.com/raspbeguy/osm
cd osm
go build -o osm ./cmd/osm
```

Go 1.26 or newer.

For a leaner binary without the TUI subcommand (drops ~10 MB of TUI deps):

```
go build -tags notui -o osm ./cmd/osm
```

The `tui` subcommand still exists in that build but prints a message telling
you to rebuild without the tag.

## First login

OAuth 2.0 has been mandatory since June 2024, so you need a client ID. Register
an application at <https://www.openstreetmap.org/oauth2/applications> with:

- Redirect URI: `http://127.0.0.1:17654/callback`
- Confidential client: no (PKCE replaces the client secret)
- Scopes: `openid`, `read_prefs`, `write_prefs`, `write_api`, `write_notes`,
  `consume_messages`, `read_gpx`, `write_gpx`

Then log in. The client id can be passed once and gets remembered:

```
osm --client-id <your-client-id> login
```

The browser opens, you approve, and the token lands in
`$XDG_CONFIG_HOME/osm/token.json` (mode 0600). After that, `osm` commands work
without `--client-id`; it lives in `$XDG_CONFIG_HOME/osm/config.json`.

The CLI talks to production by default. To target the sandbox, pass
`--api https://master.apis.dev.openstreetmap.org/api/0.6` or set `OSM_API_URL`.
The OAuth endpoints are derived from the API host, so the same `osm login`
command works against any instance.

## CLI examples

```
osm whoami
osm doctor                                 # server caps + token scopes

osm changeset list --mine
osm changeset list --mine --format '{{.ID}} {{.Comment}}'
osm changeset list --mine --format '{{json .}}'
osm changeset show 148548710
osm changeset download 148548710           # raw osmChange XML

osm edit tag node 12345 amenity=cafe name="Café Z" --comment "rename"
osm edit tag node 12345 amenity=          # empty value deletes the key
osm edit delete way 99999 --comment "obsolete"

# batch edits under a single user-managed changeset
cs=$(osm changeset open --comment "downtown survey")
osm edit tag --changeset $cs node 12345 name="Café Z"
osm edit tag --changeset $cs node 12346 amenity=bench
osm changeset close $cs

osm note create --lat 48.85 --lon 2.35 "missing footway"
osm note comment 12345 "still there"
osm note close 12345

osm message inbox
osm message read 4242
osm message delete 4242

osm trace upload run.gpx --description "morning run" --tags "run,paris"
osm trace list --format '{{.ID}} {{.Name}}'
osm trace data 9999 > backup.gpx

osm history way 12345
osm map -1.5,52.0,-1.4,52.1 > area.osm
```

`changeset list` and `message inbox|outbox` accept `--format` with a Go
`text/template`. Use `--help` on either for the field list. The helpers `json`,
`csv`, and `date` make machine-readable output one flag away (`{{json .}}` for
JSONL, `{{csv .ID .User}}` for CSV rows).

## TUI

```
osm tui
```

Browse and edit interactively. The TUI knows how to deep-link, so you can jump
straight to a screen:

```
osm tui changesets
osm tui changeset 148548710
osm tui inbox
osm tui notes
osm tui history way 12345
osm tui compose                            # build a new changeset
```

Common keys across screens: `esc` goes back, `tab` swaps focus between split
panes, `/` enters filter mode on lists, `r` refreshes. The compose flow stages
elements locally, lets you edit tags and (for relations) members in a
two-pane view, then submits everything as one atomic upload.

The TUI tries to match the terminal's light/dark mode using `COLORFGBG`. Force
a theme with `GLAMOUR_STYLE=light` or `GLAMOUR_STYLE=dark`.

## Files and environment

| Path                                | Purpose                       |
| ----------------------------------- | ----------------------------- |
| `$XDG_CONFIG_HOME/osm/token.json`   | OAuth2 access + refresh token |
| `$XDG_CONFIG_HOME/osm/config.json`  | Persisted CLI defaults        |

| Variable           | Effect                                                 |
| ------------------ | ------------------------------------------------------ |
| `OSM_CLIENT_ID`    | OAuth2 client id (or `--client-id`)                    |
| `OSM_API_URL`      | API base URL (or `--api`); auth endpoints follow host  |
| `OSM_TOKEN_PATH`   | Override token path                                    |
| `OSM_CONFIG_PATH`  | Override config path                                   |
| `GLAMOUR_STYLE`    | `light` or `dark`; overrides terminal detection        |
| `COLORFGBG`        | Read for terminal-bg detection if `GLAMOUR_STYLE` unset|

## Library usage

The libraries are usable on their own. Sketch:

```go
import (
    "context"
    "github.com/paulmach/osm"
    osmapi "github.com/raspbeguy/osm/api"
    "github.com/raspbeguy/osm/auth"
)

cfg := auth.Config{
    ClientID: "your-client-id",
    Scopes:   []string{"read_prefs", "write_api"},
}
tok, _ := auth.Login(context.Background(), cfg)
_ = auth.SaveToken(tok)

c := osmapi.NewClient(auth.HTTPClient(context.Background(), cfg, tok))

id, err := c.WithChangeset(ctx,
    osm.Tags{{Key: "comment", Value: "rename cafe"}},
    func(csID osm.ChangesetID) error {
        n, err := c.GetNode(ctx, osm.NodeID(12345))
        if err != nil { return err }
        n.Tags = append(n.Tags, osm.Tag{Key: "name", Value: "Café Z"})
        _, err = c.ModifyNode(ctx, csID, n)
        return err
    })
```

Errors are typed sentinels: `ErrConflict`, `ErrGone`, `ErrChangesetClosed`,
`ErrPreconditionFailed`, `ErrNotFound`, `ErrNilChange`. Match with
`errors.Is`. See `api/errors.go`.

`WithChangeset` opens, runs the closure, and always tries to close (even on
fn error and even if the caller's context was cancelled), so a half-submitted
upload doesn't leave a changeset open on the server.

## Man page

```
sudo install -m 0644 man/osm.1 /usr/local/share/man/man1/
mandb        # if your distro uses it
man osm
```

## What's not yet here

- Element creation from the CLI (the TUI compose flow covers this).
- PBF / XML *file writing* (use `paulmach/osm` directly).
- Overpass and Nominatim. Go clients already exist:
  `serjvanilla/go-overpass`, `philiphil/go-nominatim`.
- Moderator endpoints (block users, hide comments).

## Tests

```
go test ./...
```

Integration tests against `master.apis.dev.openstreetmap.org` are gated behind
`//go:build integration` and require a token with write scopes on the sandbox:

```
go test -tags integration ./api/...
```

