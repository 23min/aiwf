# tools/ — Go rules

These rules apply to all code under `tools/` (the Go monorepo). The repo-wide engineering principles in the root `CLAUDE.md` (KISS, YAGNI, no half-finished implementations, errors-as-findings, trace-first writes, immutability of done) cascade in on top of these.

## Formatting and linting

- **`gofumpt`** is the formatter. Run via `golangci-lint`, no separate install. Anything `gofumpt`-clean is also `gofmt`/`goimports`-clean.
- **`golangci-lint`** is the only linter. Enabled set: `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gocritic`, `revive`, `gosec`, `bodyclose`, `unconvert`, `misspell`, `gofumpt`, `goimports`. Config in `.golangci.yml` at repo root.
- CI fails on any lint finding. No `//nolint` directives without a one-line rationale comment.

## Testing

- **`testing` (stdlib) + `github.com/google/go-cmp`** for comparison-heavy assertions. No testify, no assertion DSLs.
- **Table-driven** when ≥2 cases exercise the same function. Single-case tests stay flat.
- **Subtests via `t.Run(name, ...)`** for each table case.
- **Golden files** under `testdata/` for snapshot assertions. Synthetic content only — fixtures must read as obviously fictional, not as anonymized copies of real projects.
- **Race detector on every CI run:** `go test -race ./tools/...`.

## Coverage

- **100% line coverage on `tools/internal/...` packages.** CI fails if any line is uncovered.
- **Exclusions** (intentionally small):
  - `tools/cmd/aiwf/main.go` — covered by integration tests against the binary, not unit tests.
  - Generated code.
  - Specific lines marked `//coverage:ignore <reason>`. Each occurrence reviewed in PR.
- Adding to the exclusion list requires a documented reason in the PR description.
- **Per-package coverage reported in PR description** for any new package added under `tools/internal/`. Don't ship new code at unknown coverage.

## Error handling

- Wrap every error returned across a function boundary with context: `fmt.Errorf("loading projection %s: %w", path, err)`.
- Compare errors with `errors.Is(err, ErrFoo)` and `errors.As(err, &target)`. Never `err == ErrFoo` except for sentinels you own.
- Sentinel errors (`var ErrNotFound = errors.New("not found")`) for stable conditions. Typed errors (`type ValidationError struct{...}`) when the error carries data.
- **Library code never panics or `os.Exit`s.** Only `cmd/<tool>/main.go` calls `os.Exit`.

## Concurrency

- Pass `context.Context` as the first argument of every IO-touching function.
- Use `context` for cancellation only. Don't stuff request-scoped data via `context.WithValue` except across API boundaries.
- Never hold a mutex across an IO call.
- The event log's append path holds a process-level flock; never call out to anything that might block on it from within a held flock.

## CLI conventions

Every binary (currently just `aiwf`) follows:

- **Exit codes:** `0` ok, `1` findings (validation succeeded but reported issues), `2` usage error, `3` internal error.
- **Output:** JSON by default. `--pretty` indents JSON for human reading; the unindented default is what CI scripts and downstream tools consume.
- **JSON envelope:** `{ tool, version, status, findings, result, metadata }`. `status` is one of `ok`, `findings`, `error`. `findings` is an array (possibly empty); `result` carries the verb's payload (graph subset, history slice, transition list, etc.); `metadata` carries timing, counts, and the calling correlation_id when present.
- **Logging:** `log/slog` to stderr (default level `INFO`). Tool output goes to stdout. `fmt.Fprintln(os.Stderr, …)` is not a substitute — slog gives consumers a uniform structured logging surface.
- **Flags:** `--help`, `--version`, `--pretty` plus tool-specific. No global config files; everything via flags, env, or `.ai-repo/config/<tool>.json`.
- **No package-level mutable state.** Pass dependencies via struct fields. **In particular, don't introduce production patterns purely to satisfy test-injection** — if tests need to swap a dependency, the production code uses constructor injection (`func New(deps Deps) *T`); never a package-level `var registry = map[…]` that tests mutate.

## Event log discipline

The event log is the framework's source of truth. Code that writes to it must:

- Append under the process-level flock that governs `.ai-repo/`.
- Use `O_APPEND` exclusively — never `O_TRUNC`, never seek.
- Canonicalize event payloads using RFC 8785 (JSON Canonicalization Scheme) before computing `patch_sha256` or `post_state_hash`. Don't roll your own canonicalization.
- Record the event *before* applying the effect. The confirmation event is what marks success.
- Treat `events.jsonl` as append-only forever. Compaction, if it ever lands, is a separate operation with its own design doc; no normal write path ever rewrites earlier events.

## Dependencies

- Minimize external deps. Each new dep needs a one-line justification in the PR description.
- `CGO_ENABLED=0` — binaries must be statically linked.
- `go 1.22` minimum. Bump deliberately.
- One `go.mod` for the entire `tools/` tree.

## Naming

- Package names: short, lowercase, no underscores, no plurals (`projection` not `projections`).
- Avoid stuttering: `projection.Projection` is wrong; `projection.Build` or `projection.Result` is right.
- Exported identifiers must have a doc comment starting with the identifier name.
- Acronyms stay capitalized: `parseURL`, `httpClient`, `jsonOut` — not `parseUrl`.

## Type design

- **Closed-set enums ship only used values.** When defining a closed set of constants or enum values (severity, status, kind, action), ship only the values that have a current call site. Speculative future values violate YAGNI even when they're "just constants" — they imply consumers without consumers. Add new values when the first real call site lands.
- **Boundary contracts are the source of truth for kind-specific vocabularies** — Go enums for status/kind/action are validated against `framework/modules/<name>/contracts/*.yaml` at startup, or generated from them. Don't drift the Go side from the YAML side; if they disagree, the contract YAML wins and the Go code is the bug.

## Pre-PR checklist

Before opening a PR that touches `tools/`, walk this checklist against your diff. Report conformance in the PR description.

**Architecture:**
- [ ] No new package-level mutable state. New dependencies passed via struct fields at construction.
- [ ] `context.Context` as the first arg of every new IO function (subprocess, file, network).
- [ ] No new closed-set constants without a current call site.
- [ ] Any code path that writes structural state appends an event before applying its effect.

**Errors and IO:**
- [ ] Every error returned across a function boundary wrapped with `%w` and context.
- [ ] `errors.Is`/`errors.As` for comparisons (never `==` on non-sentinel errors).
- [ ] `log/slog` for logging (not `fmt.Fprintln` to stderr).
- [ ] Library code never `panic`s or `os.Exit`s — only `cmd/<tool>/main.go` calls `os.Exit`.

**Tests and quality:**
- [ ] `go vet ./tools/...` clean.
- [ ] `golangci-lint run` clean.
- [ ] `go test -race ./tools/...` clean.
- [ ] Per-package coverage reported in PR description for any new package under `internal/`.
- [ ] Each new dep has a one-line justification in the PR description.
- [ ] No legacy project identifiers introduced (the `scrub` workflow will catch these, but check locally first).

**Docs:**
- [ ] Future integrations referenced in skill / template / changelog prose cite an open issue number.
- [ ] `CHANGELOG.md` `[Unreleased]` updated for any user-visible change. Skip only for internal-only refactors with no observable effect.

If a rule needs to be relaxed, propose the change in the same PR — don't silently violate.
