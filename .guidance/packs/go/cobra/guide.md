# When working in Go

**Version & house style:** read `go.mod` for the module's Go version and use
version-appropriate idioms; the repo's own conventions win.

## Toolchain
- **`gofumpt`** (via `golangci-lint`, no separate install) formats; gofumpt-clean implies
  gofmt/goimports-clean.
- **`golangci-lint`** is the linter. Solid enabled set: `errcheck`, `govet`, `staticcheck`,
  `ineffassign`, `unused`, `gocritic`, `revive`, `gosec`, `bodyclose`, `unconvert`,
  `misspell`. No `//nolint` without a one-line rationale.

## Testing
- Stdlib `testing` + `github.com/google/go-cmp`. No testify or assertion DSLs.
- Table-driven when ≥2 cases share a function; subtests via `t.Run`; golden files under
  `testdata/`. Race detector on every CI run (`go test -race ./...`).
- `t.Parallel()` first line of parallelizable tests; seed process-wide fixtures once in
  `TestMain` with `os.Setenv` (never `t.Setenv` under `t.Parallel` — it panics).

## Errors
- Wrap across boundaries: `fmt.Errorf("loading %s: %w", path, err)`. Compare with
  `errors.Is` / `errors.As`, never `==` (except sentinels you own).
- Sentinel errors for stable conditions; typed errors when the error carries data.
- Library code never `panic`s or `os.Exit`s — only `main` exits.

## Idioms
- **Accept interfaces, return concrete types.** Define small interfaces at the consumer, not
  the producer.
- `context.Context` is the first arg of every IO-touching function (cancellation only);
  never hold a mutex across an IO call.
- `defer` for cleanup; check the error on a deferred `Close` via a named return.
- `errgroup` (or `sync.WaitGroup`) for fan-out; no package-level mutable state — inject
  dependencies via struct fields / constructors.
- Closed-set enums ship only the values with a call site today. Acronyms stay capitalized
  (`parseURL`). `CGO_ENABLED=0` for static binaries.

## CLI
- `cobra` for CLIs. Exit codes: `0` ok, `1` findings, `2` usage, `3` internal. `log/slog` to
  stderr; tool output to stdout. Human-readable default, `--format=json` secondary.
