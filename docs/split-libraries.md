# Splitting the libraries into their own repo

The libraries (`api/`, `auth/`, `internal/osmchange/`) and the binary
(`cmd/osmctl/`) live in one repo today. If an external consumer ever asks for a
narrower `go.mod`, here's what the split would cost.

## Mechanical work

Roughly half an afternoon.

1. New module `github.com/raspbeguy/osm-client` (name TBD).
2. Move `api/`, `auth/`, and `internal/osmchange/` into it as the module root.
3. Tag `v0.1.0`.
4. In this repo, replace those directories with `go get`+imports.
5. `internal/version/` stays here; the binary owns its versioning.

Tests carry over unchanged. The module path stays
`github.com/raspbeguy/osm` for the binary repo.

## Design pass

The only real friction. Two spots have binary concerns leaking into the lib:

- `auth/store.go` bakes `$XDG_CONFIG_HOME/osm/` into `tokenPath()` /
  `configPath()`. A library shouldn't make user-facing path decisions. Push
  this into the binary: `LoadToken(path)` / `SaveToken(path, tok)` take a
  path argument; the binary owns the default.
- `auth/Login` ships with `pkg/browser` and a localhost listener. That's a
  CLI UX, not a primitive. Split into lower-level helpers (`AuthCodeURL`,
  `Exchange`, `Verifier`) and let the binary do the browser dance and the
  HTTP callback.

After that, the library surface is purely the API client and OAuth2
primitives. Nothing else in the codebase has cross-layer coupling.

## Ongoing cost

A few minutes per cross-cutting change:

- PR in `osm-client`, tag a release, bump `go.mod` here, PR here.
- Doubled CI.
- Issue triage gains a "lib or binary?" routing step.

## Why wait

The codebase is already split-ready: clean directory boundaries, no library
code importing binary code, context-first APIs, sentinel errors. The split
won't force an API redesign. So the cost is mostly setup overhead, not
refactor pain. Wait until an external consumer actually asks, then do it
once.
