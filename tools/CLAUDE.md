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

### Substring assertions are not structural assertions

A test that greps for a literal in rendered output (HTML, Markdown, JSON) proves the literal exists *somewhere*. It does not prove the literal is in the right *place*. The right place is what the user (or the next renderer) actually consumes; the literal floating in the wrong section is still a bug that ships.

Concrete shape:
- For HTML output, parse the document with `golang.org/x/net/html` (or an equivalent) and assert presence inside the named `<section>`, attribute, or descendant chain. A standalone substring match is acceptable only when the value is unique and the location is irrelevant (e.g., a stable token in a JSON envelope).
- For markdown output, walk the heading hierarchy and assert the prose appears under the expected section, not just on the page.
- For multi-tab / multi-section pages, every substring assertion must name *which* section it expects the value in. "AC anchor exists" is not enough; "AC anchor exists inside `data-tab=manifest`" is.

If the literal under test is short or generic enough to plausibly appear in unrelated places (e.g. an id="ac-1" attribute, the word "strict", a status name like "active"), assume it does and use a structural assertion.

Why this rule exists: I3 step 5 shipped milestone-page tests that asserted `id="tab-overview"`, `href="#ac-1"`, `policy-strict">strict` etc. as plain substring matches. Two of those would have passed even with the AC rendered in the wrong tab, the policy badge swapped, or the anchor wired backwards. The user caught this in audit; the tests were structurally weak from the start.

### Render output must be human-verified before the iteration closes

Test suites pin code correctness — they do not pin *feature* correctness. For UI / rendered output (HTML pages, generated docs, status outputs that the user reads), running the binary against a real fixture and visually inspecting the result is part of "done," not an optional follow-up. A green test suite says "no regressions in what we asserted"; only a manual look says "the page actually communicates what it should."

Concrete shape:
- Before claiming a render-iteration step closed, render against a non-trivial real fixture (the kernel repo's own planning tree is the canonical one), open the result, exercise the interactive surface (every tab, every link, every conditional content path), and only then mark the step done.
- If you cannot run the binary in your environment (sandbox, CI-only), say so explicitly to the user instead of declaring success — the tests do not stand in for that pass.
- An end-to-end golden snapshot of one full page (HTML byte-equal to a known-good fixture) is a good auxiliary safety net, but it doesn't replace the human look-through. Snapshots only catch *changes*; they don't catch "this was wrong on day one."

Why this rule exists: I3 step 5 shipped six milestone tabs with placeholder Build/Tests/Provenance content paths that no test exercised. The rendered output was never opened in a browser. A green `go test ./...` was treated as completion; the user's audit caught the gap.

### Test untested code paths before declaring code paths "done"

When a function has a branch (a `switch`, an `if`, a filter), every reachable branch must have a test that traverses it — or the branch must be marked `//coverage:ignore` with a one-line rationale. "Tests pass" with code paths not exercised is "tests pass for the paths I happened to think about."

Concrete shape:
- Before committing a feature, run `go test -coverprofile=cov.out ./<pkg>/...` and skim the uncovered lines. Each uncovered line is either: (a) a missing test (write it), (b) defensive code that can't fire (delete it), or (c) genuinely unreachable in production (mark it `//coverage:ignore <reason>`).
- For typed view-builders that filter or branch on input (e.g. "is this a phase event?", "does this commit have an authorize trailer?"), the test set must include at least one input that takes each branch. A fixture with no scopes, no phase events, no force trailers exercises only the empty-state branch — that's not coverage of the populated path.
- When the package gains a new typed input (a new trailer, a new field on a struct), audit the consumers' branches the same way: which call sites now have an unexercised arm? Write the missing test before the next commit.

Why this rule exists: I3 step 5's `phaseEventsFromHistory`, `firstTestsTrailer`, `provenanceFor`, and `linkedEntitiesFor` were all wired in but never exercised by any test fixture that produced phase history, test trailers, scopes, or cross-kind references. The functions could have returned wrong shapes silently and nothing would have failed.

### Don't paper over a test failure — root-cause it

When a test fails in a way that doesn't match its premise, the failure is information about the system, not about the test. Working around it (changing the test setup until it passes, adding manual git commits the production path doesn't make, sleeping until a race resolves) leaves the original signal unread. The test now passes for a reason other than what it was supposed to verify.

Concrete shape:
- If a test fails with a state error (lock contention, projection mismatch, "not found"), the first action is to dump the state at the point of failure (`t.Logf` the on-disk content, the trailer set, the lock holder) and read the actual cause. Only after you understand it should you decide whether the test or the production code needs to change.
- A "manual git commit to keep things clean" inside a test is a yellow flag — the production verb is not making that commit and the user won't either. Either the verb should make it (production bug) or the test fixture should be set up so it isn't needed (test bug); not both.
- "Workaround applied; investigation deferred" comments are owed an issue / gap entry; otherwise they accumulate as silent debt.

Why this rule exists: I3 step 2A's `TestRun_AddACWithTestsFlag` originally hit a verb error after a hand-edit; I added a manual `git add -A && git commit` to make the test pass without diagnosing why the verb's projection ran in the wrong direction. The test now passes for a different reason than its assertion claims.

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

## Release process

Releases of `aiwf` are git tags on `poc/aiwf-v3` of the form `vX.Y.Z`. The Go module proxy resolves them when a consumer runs `aiwf upgrade` or `go install <pkg>@latest`. There is no separate release artifact to publish, but the user-facing changelog must stay in step.

Before tagging `vX.Y.Z`:

1. In a single release-prep commit, edit [`CHANGELOG.md`](../CHANGELOG.md):
   - Rename the `## [Unreleased]` heading to `## [X.Y.Z] — YYYY-MM-DD`.
   - Add a fresh empty `## [Unreleased]` heading at the top (above the new version section).
   - Verify the moved entries summarize the user-visible delta — gaps closed, verbs added, behavior changes. Internal refactors that change nothing observable can be omitted.
2. Use commit subject `release(aiwf): vX.Y.Z`.
3. Push the commit, then `git tag vX.Y.Z` pointing at it, then `git push origin vX.Y.Z`.

Skipping the changelog edit means the tag-push CI check fails: the workflow at [`.github/workflows/changelog-check.yml`](../.github/workflows/changelog-check.yml) verifies that every pushed `v*` tag is reachable from a commit whose `CHANGELOG.md` contains a matching `## [X.Y.Z]` heading. Per the kernel's "framework correctness must not depend on the LLM's behavior" rule, the check is the guarantee — the human-facing rule above is just the convenient version.

Patch releases that are pure-mechanical (e.g. a `go.sum` refresh with no behavior delta) still require a CHANGELOG entry, even if it is a single line saying "no functional changes" — the workflow does not distinguish empty from missing.

## Dependencies

- Minimize external deps. Each new dep needs a one-line justification in the commit message or PR description.
- `CGO_ENABLED=0` — binaries must be statically linked.
- `go 1.24` minimum. Bump deliberately. Last bumped from 1.22 → 1.24 in G43; rationale on file.
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

## What's enforced and where

The kernel's "framework correctness must not depend on LLM behavior" principle applies here too: the rules below are enforced by tooling at named chokepoints, not by remembering to tick a checklist. This section names the chokepoint for each rule so a contributor (human or LLM) can see what will block a bad commit and what is still advisory.

| Rule                                                         | Chokepoint                                                       | Status                  |
|--------------------------------------------------------------|------------------------------------------------------------------|-------------------------|
| `gofmt` / `goimports` / `gofumpt` clean                      | `golangci-lint run` (formatters block) — CI `lint` job           | Blocking via CI         |
| Lint set passes (`errcheck`, `govet`, `staticcheck`, …)      | `golangci-lint run` — CI `lint` job                              | Blocking via CI         |
| `go vet` clean                                               | `go vet ./tools/...` — CI `vet` job                              | Blocking via CI         |
| Tests pass with race detector                                | `go test -race ./tools/...` — CI `test` job                      | Blocking via CI         |
| Build succeeds (`CGO_ENABLED=0`)                             | `go build` — CI `build` job                                      | Blocking via CI         |
| End-to-end verb regressions                                  | `aiwf doctor --self-check` — CI `selfcheck` job (G9)             | Blocking via CI         |
| Vulnerable transitive deps                                   | `govulncheck ./tools/...` — CI `vuln` job (G43)                  | Blocking via CI         |
| Library code does not `panic` or `os.Exit`                   | `forbidigo` (G43)                                                | Blocking via CI lint    |
| Test helpers call `t.Helper()`                               | `thelper` (G43)                                                  | Blocking via CI lint    |
| Errors compared with `errors.Is`/`As`, wraps use `%w`        | `errorlint` (G43)                                                | Blocking via CI lint    |
| Planning-tree shape (no stray files under `work/`)           | `aiwf check --shape-only` — pre-commit hook (G41)                | Blocking pre-commit     |
| Full planning-tree validation (refs, ids, FSM, contracts)    | `aiwf check` — pre-push hook                                     | Blocking pre-push       |
| Repo-specific invariants (trailer keys, sovereign acts, etc.) | `tools/internal/policies/` — runs as a Go test package           | Blocking via CI test    |
| `context.Context` as first arg of new IO function            | Code review                                                      | Advisory                |
| No new package-level mutable state                           | Code review                                                      | Advisory                |
| Each new dep has a one-line justification                    | Code review (commit message / PR description)                    | Advisory                |
| Mutating-verb commits carry `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers | `tools/internal/policies/trailer_keys.go` + the `principal_write_sites` policy + the untrailered-entity audit (G24, G31, G32) | Blocking via CI test    |
| Bumping the Go floor in `go.mod`                             | Deliberate decision; document rationale in commit message        | Advisory                |

The four advisory lines are the items where mechanical enforcement is either too noisy (context.Context first arg — generics make a literal regex unreliable), too contextual (package-level mutable state — sometimes legitimate behind a guard), or self-policing (dep justification, deliberate floor bumps). Reviewers and `tools/CLAUDE.md` itself are the chokepoint there.

If a blocking rule needs to be relaxed for a specific call site, route it through the linter's allowlist (e.g., `forbidigo` exclusion for the verb/apply.go re-panic site) with a one-line rationale, not a `//nolint:rule` directive without explanation.
