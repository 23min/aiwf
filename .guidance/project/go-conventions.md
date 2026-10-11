# Go conventions

## Formatting and linting

- **`golangci-lint`** is the only linter. Enabled: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocritic`, `revive`, `gosec`, `bodyclose`, `unconvert`, `misspell`, `gofumpt`, `goimports`. CI fails on any finding. No `//nolint` without a one-line rationale.
- **Never `var _ = x` to silence `unused`.** A blank-identifier alias kept solely to stop `unused`/`staticcheck` flagging reserved-for-future code defeats the check whose job is to find exactly that. No caller → not ready to land; delete it and re-add with its first real caller (the shipped form of this force is `wf-codebase-health` H2).

## Concurrency

- `context.Context` is the first arg of every IO-touching function (cancellation only; don't stuff request data via `WithValue` except across API boundaries).

## CLI conventions

- **`github.com/spf13/cobra`.** Each verb is a `cobra.Command` with a `runXCmd(...)` body called from `RunE`; wire from `newRootCmd`. Closed-set flag values bind completion via `cobra.FixedCompletions(...)`; entity-id flags use `completeEntityIDFlag(kind)`. Chokepoint: `internal/cli/integration/completion_drift_test.go`.
- **Exit codes:** `0` ok, `1` findings, `2` usage, `3` internal. Verbs return `int`; the `RunE` adapter shuttles via `*exitError`.
- **Output:** human-readable by default; `--format=json` emits `{ tool, version, status, findings, result, metadata }` (`status` ∈ `ok`/`findings`/`error`); `--pretty` indents; tool output → stdout. **Diagnostic/operator logging** is opt-in and default-off ([the logging ADR](../../docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md)): `log/slog`, configured via `AIWF_LOG`/`AIWF_LOG_FORMAT`/`AIWF_LOG_FILE` (env beats `aiwf.yaml`'s `logging:` block beats the default), writes structured records to a daily-rotated file under the XDG state home by default — never stderr unless explicitly configured there. `forbidigo` (backed by the `logging-chokepoint` AST policy, which also catches `fmt.Fprintln`/`fmt.Fprintf` to `os.Stdout`/`os.Stderr`) bans bare `fmt.Print*` call sites outside the sanctioned `cliutil` text wrappers, so operator-facing text can't silently drift back onto ad hoc prints; diagnostic logging goes through `log/slog` in `internal/logger`, never a bare print.
- **Flags:** `--help`, `--version`, `-v`, `--pretty` plus verb-specific. No global config files; config via flags, env, or `aiwf.yaml`.

## Dependencies

Minimize external deps (one-line justification per new one). `CGO_ENABLED=0` (static binaries). `go 1.24` minimum in `go.mod` (the consumer-compatibility floor); CI toolchain is pinned separately to `go-version: "1.25"` (bump CI for stdlib CVEs, bump the floor only for a needed language feature). One `go.mod` for the module.

## Naming

Package names short, lowercase, singular (`entity` not `entities`). Avoid stuttering (`entity.Load`, not `entity.Entity`). Exported identifiers get a doc comment starting with the name. Acronyms stay capitalized (`parseURL`, `jsonOut`).
