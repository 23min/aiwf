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

### Test the seam, not just the layer

When a new helper, package, or shared function is wired into an existing caller (verb, dispatcher, hook), the test set must cover **both** the helper's behavior *and* the seam where it integrates. A unit test of the helper alone is necessary but not sufficient — it doesn't catch the case where the caller has a parallel source of truth and never adopts the helper.

Concrete shape: for a new verb-level helper, write at least one test that drives the verb's dispatcher (`run([]string{"<verb>", ...})`) and asserts the output reflects the helper's contract. For a check-rule helper, write a fixture-tree test that exercises the rule through `check.Run`. Test names should make the seam explicit (`TestRunVersion_UsesBuildInfoFallback`, not just `TestResolvedVersion`).

When a verb's output depends on values that only exist in a real binary — `runtime/debug.ReadBuildInfo`, `-ldflags`-stamped globals, `os.Args[0]`, `os.Executable()` — a unit test running under `go test` cannot exercise the production path. Add a binary-level integration test that builds the cmd to a tempfile and runs it as a subprocess: `go build -o $TMP/aiwf ./tools/cmd/aiwf && exec.Command($TMP/aiwf, "version")`. The cost is a few seconds per CI run; the alternative is the bug shipping.

Why this rule exists: v0.1.0 shipped with `aiwf version` returning `"dev"` even though the new `version.Current()` helper returned the correct buildinfo value. The unit test of `version.Current()` was clean. The verb still printed an unrelated package-global. Two parallel sources of truth coexisted; tests covered only the new one.

### Contract tests for upstream-cached systems

For any external system with caching semantics — HTTP proxies (the Go module proxy is the canonical example), DNS, CDN-fronted APIs — tests must pin "did we ask the right question," not just "did we parse the answer correctly."

Concrete shape: a real-system integration test (gated under `-short` so CI without network can skip) that derives the expected value through an **independent** code path, not from the same endpoint the implementation uses. For the module proxy this means: if the implementation resolves "latest" via `/@v/list`, the test independently fetches `/@v/list`, computes the expected highest semver, and asserts the implementation returns the same value. A test that just asserts "the implementation returned a non-empty version" is parsing-coverage, not resolution-correctness.

When you discover the right endpoint by reading the upstream tool's source (e.g., the Go toolchain's resolver), document that decision in a comment at the call site so future readers don't re-litigate the choice.

Why this rule exists: v0.1.0's `version.Latest()` queried the proxy's `/@latest` endpoint, which is cached separately from `/@v/list` and can serve stale pre-tag pseudo-versions for hours after a tag lands. The unit tests served whatever JSON the implementation expected and never asked whether the chosen endpoint was the right one. The Go toolchain uses `/@v/list`-first for exactly this reason — documented behavior we re-learned by failing in production.

### Spec-sourced inputs for upstream-defined input spaces

When test cases enumerate an upstream-defined input grammar — semver shapes, RFC fields, error-code families, on-disk format variants — the test must cite the spec and cover the full enumerated space, not "the example I had in mind."

Concrete shape: prefix the test data with a comment pointing at the canonical spec (e.g., `// per https://go.dev/ref/mod#pseudo-versions`), then list every case the spec defines. If you cannot cite a single source for the input space, the space isn't pinned and the tests are example-driven; either find the spec or document the omission explicitly as a known limitation.

Why this rule exists: v0.1.0's pseudo-version regex initially only matched the basic `v0.0.0-DATE-SHA` form. The Go module spec defines three shapes (basic, post-tag, pre-release-base); VCS stamping adds the `+dirty` suffix. Smoke tests caught the gaps mid-implementation. A spec-sourced test pass at design time would have exercised all four cases on the first commit.

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

## Designing a new verb

Before adding a verb to `cmd/aiwf/`, the design isn't done until you can answer **"what verb undoes this?"** Acceptable answers:

- *Another invocation of the same verb with different inputs.* Most state-transition verbs reverse this way (e.g. `aiwf promote E-01 active` undoes `aiwf promote E-01 done` if the kind allows it).
- *An explicit terminal-state transition.* `aiwf cancel`, `aiwf reallocate` (renumbers; the old id's history terminates with the rename event).
- *"You can't, and that's deliberate — here's why."* `aiwf init` is one-shot; `aiwf import` for already-present ids needs `--on-collision`. The reason gets written down.
- *"You'd open a new entity for the inverse."* Bug-fix-style reversals (e.g., add a hotfix milestone) belong here.

Not acceptable: *"we'll figure that out later"* — the verb isn't ready. See [docs/pocv3/design/design-lessons.md](../docs/pocv3/design/design-lessons.md) §"On reversal" for the principle this comes from.

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
