# Project conventions

## Style

Comments only when absolutely necessary, and only to explain the **why**, never the **what**. The code already says what it does. One line max. No multi-line block comments, no paragraph docstrings.

The one exception: doc comments on exported library identifiers when they convey a non-obvious contract (preconditions, side effects, ownership, allocation). Skip them on self-explanatory exports even if `golint` complains.

Never use em-dashes anywhere: code, comments, commit messages, docs, issue text, PR descriptions, README, error messages, anything written. Use periods, colons, commas, semicolons, or parens instead.

All public-facing prose reads like a person wrote it. Skip the AI tells: stilted hedging, parade-of-bullets where flowing sentences fit, over-structured headings, formulaic openings, reflexive politeness padding. Plain, direct, casual when casual fits.

`gofmt -s` and `staticcheck` must be clean before any commit.

## Library design

Reuse `paulmach/osm` types (`osm.Node`, `osm.Way`, `osm.Relation`, `osm.Tags`, `osm.Change`). Don't redefine them; our libraries layer authenticated transport and write workflows on top.

Every API method takes `context.Context` as its first parameter.

Errors are typed sentinels (`ErrConflict`, `ErrGone`, `ErrChangesetClosed`, `ErrPreconditionFailed`, ...) wrapped with `fmt.Errorf("...: %w", err)`. Callers match with `errors.Is`, never with string compares.

Never log OAuth tokens. Redact tokens in any debug output.

## Testing

Unit tests use `httptest.Server` with captured OSM API fixtures. No real network in default test runs.

Integration tests against `master.apis.dev.openstreetmap.org` are gated behind `//go:build integration` and excluded from `go test ./...`.

## Dependencies

Keep the dependency surface small: `paulmach/osm`, `golang.org/x/oauth2`, `spf13/cobra`, `pkg/browser`. Any new dep needs justification.

## API base URL

Default: production (`https://api.openstreetmap.org/api/0.6`).

Sandbox (`https://master.apis.dev.openstreetmap.org/api/0.6`) opt-in via `--api` flag or `OSM_API_URL` env.

## Git workflow

Local repo config mirrors `~/repo/terrain` (personal identity, signed commits):

```
user.name = Guy Godfroy
user.email = guy.godfroy@gugod.fr
user.signingkey = 9CAFADF955878D514497B16EE4B5C3548E3CFB30
commit.gpgsign = true
tag.gpgsign = true
```

All set as **local** config (`git config ...`, never `--global`).

Commit frequently, but only when the working state compiles, tests pass, and the feature at hand works. No WIP commits on `main`.

Commit messages are **one line**, Conventional Commits style: `<type>(<scope>): <subject>`.

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `build`, `ci`. Scope is the package (`auth`, `api`, `cmd`, `osmchange`), or omitted for repo-wide changes. Subject is imperative, lowercase, no trailing period, ≤72 chars total.

Examples:

```
feat(auth): add pkce login with localhost callback
fix(api): handle 412 on changeset version mismatch
refactor(api): extract changeset upload into osmchange/
chore: add staticcheck to ci
```

No multi-line bodies unless a non-obvious *why* needs recording. No co-author trailers unless explicitly asked.
