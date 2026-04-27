# tools/ — Go rules

These rules apply to all code under `tools/` (the Go monorepo). The repo-wide engineering principles in the root `CLAUDE.md` (KISS, YAGNI, no half-finished implementations, errors-as-findings) cascade in on top of these.

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

- **High coverage on `tools/internal/...` packages.** PoC target is 90%; failing checks for low coverage are advisory at this stage.
- **Exclusions** (intentionally small):
  - `tools/cmd/aiwf/main.go` — covered by integration tests against the binary, not unit tests.
  - Generated code.
  - Specific lines marked `//coverage:ignore <reason>`.
- The PoC is small enough that 100% coverage on internal packages is realistic; aim for it but don't block on it.

## Error handling

- Wrap every error returned across a function boundary with context: `fmt.Errorf("loading frontmatter from %s: %w", path, err)`.
- Compare errors with `errors.Is(err, ErrFoo)` and `errors.As(err, &target)`. Never `err == ErrFoo` except for sentinels you own.
- Sentinel errors (`var ErrNotFound = errors.New("not found")`) for stable conditions. Typed errors (`type ValidationError struct{...}`) when the error carries data.
- **Library code never panics or `os.Exit`s.** Only `cmd/<tool>/main.go` calls `os.Exit`.

## Concurrency

- Pass `context.Context` as the first argument of every IO-touching function.
- Use `context` for cancellation only. Don't stuff request-scoped data via `context.WithValue` except across API boundaries.
- Never hold a mutex across an IO call.

## CLI conventions

Every binary (currently just `aiwf`) follows:

- **Exit codes:** `0` ok, `1` findings (validation succeeded but reported issues), `2` usage error, `3` internal error.
- **Output:** Human-readable text by default; `--format=json` emits a structured JSON envelope for CI scripts and downstream tools. `--pretty` (with `--format=json`) indents the envelope. `aiwf` is an interactive CLI first; the JSON shape is the secondary surface.
- **JSON envelope:** `{ tool, version, status, findings, result, metadata }`. `status` is one of `ok`, `findings`, `error`. `findings` is an array (possibly empty); `result` carries the verb's payload; `metadata` carries timing, counts, and the calling correlation_id when present.
- **Logging:** `log/slog` to stderr (default level `INFO`). Tool output goes to stdout. `fmt.Fprintln(os.Stderr, …)` is not a substitute.
- **Flags:** `--help`, `--version`, `--pretty` plus verb-specific. No global config files; everything via flags, env, or `aiwf.yaml` at the consumer repo root.
- **No package-level mutable state.** Pass dependencies via struct fields. In particular, don't introduce production patterns purely to satisfy test-injection — if tests need to swap a dependency, the production code uses constructor injection (`func New(deps Deps) *T`); never a package-level `var registry = map[…]` that tests mutate.

## Commit conventions

Every mutating verb writes a structured trailer in its commit message so `aiwf history` can render per-entity timelines:

```
aiwf-verb: promote
aiwf-entity: M-001
aiwf-actor: human/peter
```

Commit subject lines follow Conventional Commits (`feat(plan): ...`, `chore(plan): ...`, `docs(adr): ...`).

## Dependencies

- Minimize external deps. Each new dep needs a one-line justification in the commit message or PR description.
- `CGO_ENABLED=0` — binaries must be statically linked.
- `go 1.22` minimum. Bump deliberately.
- One `go.mod` for the entire `tools/` tree.

## Naming

- Package names: short, lowercase, no underscores, no plurals (`entity` not `entities`).
- Avoid stuttering: `entity.Entity` is wrong; `entity.Load` or `entity.Result` is right.
- Exported identifiers must have a doc comment starting with the identifier name.
- Acronyms stay capitalized: `parseURL`, `httpClient`, `jsonOut` — not `parseUrl`.

## Type design

- **Closed-set enums ship only used values.** When defining a closed set of constants or enum values (status, kind, action), ship only the values that have a current call site. Speculative future values violate YAGNI even when they're "just constants."
- **The PoC's six kinds** (epic, milestone, ADR, gap, decision, contract) and their **status sets** are hardcoded in Go for the PoC. They are intentionally not driven by external YAML — that move is deferred until a real consumer needs to customize the vocabulary.

## Pre-commit checklist

Before committing on the PoC branch:

- [ ] `go vet ./tools/...` clean.
- [ ] `golangci-lint run` clean.
- [ ] `go test -race ./tools/...` clean.
- [ ] No new package-level mutable state.
- [ ] `context.Context` as the first arg of every new IO function.
- [ ] Each new dep justified.
- [ ] If the commit is a mutating verb, structured `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailer is present.

If a rule needs to be relaxed, decide deliberately and note it in the commit message — don't silently violate.
