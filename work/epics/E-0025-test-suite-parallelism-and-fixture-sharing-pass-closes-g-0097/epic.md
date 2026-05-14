---
id: E-0025
title: Test-suite parallelism and fixture-sharing pass — closes G-0097
status: active
---
# E-0025 — Test-suite parallelism and fixture-sharing pass (closes G-0097)

## Goal

Convert the Go test suite from serial-with-per-test-setup to parallel-with-shared-fixtures across the load-bearing packages, so `make test` and CI's `go test` job complete in a fraction of today's wall time. The change is mechanical (a pattern, applied per package) and the spike on `spike/test-parallel` proved the pattern works on the largest single package (`internal/verb/`: ~4× faster non-race, ~2.4× with race).

## Context

G-0097 documents the discovery: the suite is slow not because individual tests are slow, but because almost no test calls `t.Parallel()`, every test re-does its own git-init / tree-load / binary-build, and `-race` at default parallelism flakes under the resulting subprocess fan-out. The pattern is uniform across packages — the fix is uniform too.

The kernel's test-discipline rules in CLAUDE.md (`-race` on every CI run, "framework correctness must not depend on the LLM's behavior", "test the seam, not just the layer") are unaffected by this epic — none of them constrain *how* tests are scheduled, only *what* they assert. The race detector stays on; this epic just lets it run faster.

The spike's secondary finding — `-race -parallel=GOMAXPROCS` flakes under heavy git-subprocess fan-out on macOS — is environmental, not a real data race. Capping `-parallel` is the kernel of the fix; the cap belongs in `Makefile` and `.github/workflows/go.yml` so CI and local runs agree.

## Scope

### In scope

- **Apply the `TestMain` + `t.Parallel()` pattern across `internal/*` test packages.** For each package: a `setup_test.go` with `TestMain` that seeds the GIT_AUTHOR/COMMITTER identity once via `os.Setenv`; `t.Setenv` removed from per-test helpers; `t.Parallel()` added to every top-level test that doesn't already call `t.Setenv`/`t.Chdir` or guard a global. Same shape as the spike. Packages in scope: `internal/verb`, `internal/check`, `internal/initrepo`, `internal/htmlrender`, `internal/policies`, `internal/aiwfyaml`, `internal/contractverify`, `internal/tree`, `internal/gitops`, `internal/skills`, `internal/config`, plus any others discovered during the milestone.
- **Apply the same pattern to `cmd/aiwf/` test files.** Audit each file individually because some tests intentionally rely on subprocess isolation (`runBin`, `aiwfBinary`); those still benefit from `t.Parallel` since they each `t.TempDir`. Tests that mutate process-level state (env, `os.Args`) skip parallelism.
- **`sync.Once` the no-ldflags build in `cmd/aiwf/binary_integration_test.go`.** 5 of 7 tests build a non-stamped binary; share one. The 2 ldflags-stamped tests still build their own. Pattern matches the existing `aiwfBinary` in `cmd/aiwf/integration_test.go`.
- **Memoize the live-repo `tree.Load(repoRoot)` in `internal/policies`.** A `TestMain` (or a `sync.Once`-guarded helper) loads the repo tree once and shares the `*Tree` across `TestPolicy_ThisRepoTreeIsClean`, `TestPolicy_ThisRepoDriftCheckClean`, `internal/policies/m080_test.go::loadM080Spec`, and any other consumers. Tests must not mutate the shared tree — assert this via review.
- **Cap `-race` parallelism in CI and local Makefile.** `Makefile`'s `test-race` target gains `-parallel 8`; `.github/workflows/go.yml`'s test job gains the same. Document the rationale (race + heavy subprocess fan-out flakes on macOS; CI Linux less affected but the cap is uniform). One-line change in each file.
- **Add a `setup_test.go`-presence policy test under `internal/policies/`.** The test walks every `internal/*` directory containing `*_test.go` and asserts each has a `setup_test.go` declaring a `TestMain`. AST-level check, not substring. Ships in M-C alongside the CLAUDE.md `## Test discipline` section so the rule and its chokepoint land together. No per-function `t.Parallel()` audit — too pedantic for the marginal value; presence of `setup_test.go` is a reasonable proxy.

### Out of scope

- **Fixture snapshotting for `internal/htmlrender/htmlrender_test.go::writeFixtureTree`.** The inline-string fixture is a small per-test cost; converting it to `testdata/` is a clean win but lower priority than the parallelism pass and best done as a follow-up if wall-time still feels high.
- **Pre-baked `aiwf init`-ed skeleton tempdir for `cmd/aiwf/` tests.** Snapshot-copy via `os.CopyFS` would save the per-test `aiwf init` cost, but the implementation is non-trivial (must include `.git` correctly, must not leak between runs) and the parallelism pass alone may make it unnecessary. Defer.
- **Topology sharing across `cmd/aiwf/integration_g37_test.go`'s 11 tests.** Each test's bare-origin + N-clone setup has enough variation that sharing is real refactor work. Plain `t.Parallel` adoption is the right scope here; topology sharing is deferred.
- **Moving `runBin` callers to in-process `run([]string{...})`.** Tempting but risky — subprocess isolation is load-bearing in many tests (exit codes, stdout/stderr capture, env isolation, the kernel's "test the actual binary" stance per CLAUDE.md). Touching this in bulk would be a separate, much-larger epic.
- **Dropping `-race` or `-coverprofile`.** Both are deliberate kernel-correctness commitments per CLAUDE.md.
- **Investigating the macOS `git add: signal: segmentation fault` under heavy parallel fan-out.** `-parallel 8` (or below) avoids it reliably; chasing the upstream-environment cause would be a yak-shave.
- **A per-function `t.Parallel()` policy check.** Auditing every `Test*` body for `t.Parallel()` with a clean opt-out story (filename convention, directive comments) creates more friction than it prevents. The `setup_test.go`-presence check above is the chokepoint; per-function discipline lives in CLAUDE.md and review.
- **Shipping the convention to downstream consumers** (e.g., via a `wf-rituals` skill). Consumers either copy aiwf's `CLAUDE.md ## Test discipline` section into their own `CLAUDE.md` or wait for a follow-up gap to propose an opt-in skill — no obligation imposed by this epic. The author of this epic is also a consumer; the consumer-copy path is the working assumption.

## Constraints

- **No test semantics change.** `t.Parallel()` adoption must not change what tests assert. Race-detector findings, `-coverprofile`, and the existing assertion structure all survive the conversion. Reviewer chokepoint: a converted test that newly passes for a different reason than its assertion claims (per CLAUDE.md "Don't paper over a test failure") fails review.
- **TDD: not required.** This is pure test-infrastructure refactor; no new production logic, no FSM changes. Each milestone's gate is "the same suite (or its converted subset) still passes after the change, including under `-race -parallel 8`."
- **One commit per milestone — except M-A.** M-A explicitly relaxes this to one commit per per-package conversion plus a leading cap-change commit. The milestone is refactor-shaped, not mutation-shaped: per-package commit signal (which package, what shape, did `-race -count=10` survive afterward) is more valuable than atomic-milestone signal. Each commit still carries the standard trailers (`aiwf-verb`, `aiwf-entity: M-NNNN`, `aiwf-actor`). M-B and M-C remain single-commit.
- **Forward-compatibility with future test additions.** The `TestMain` pattern must not introduce package-level mutable state that a future test could surprise itself with. `os.Setenv` for the GIT identity is intentional (process-wide constant; never mutated); other patterns (e.g., a shared `*Tree`) get a single `sync.Once`-guarded loader and a comment saying "do not mutate."
- **AI-discoverability.** A new `## Test discipline` section in `CLAUDE.md` documents the pattern (`TestMain` for env, `t.Parallel` default-on, shared-fixture rule), so a future contributor or AI assistant authoring a new test file picks up the convention by reading the playbook rather than the existing code. The CLAUDE.md change is paired in the same commit (M-C) with the `setup_test.go`-presence policy test, so the rule and its chokepoint ship together.
- **What undoes this?** Re-running the conversion script with the inverse rules (or `git revert`) — the pattern is mechanical, the inverse is mechanical. No durable state is created.

## Success criteria

- [ ] `go test ./internal/...` wall time at default parallelism drops measurably from baseline (target: ≥2× faster on the same hardware as the baseline run; record both numbers in the wrap-up).
- [ ] `go test -race -parallel 8 ./...` is reliable across 10 consecutive runs (no flake, no timeout). Documented in the wrap-up with the run log.
- [ ] Every package under `internal/*` has a `setup_test.go` (or equivalent) declaring its `TestMain` + parallel-by-default convention.
- [ ] `cmd/aiwf/binary_integration_test.go` shares the no-ldflags build via `sync.Once`; the test count passes unchanged.
- [ ] `internal/policies/` shares the live-repo `tree.Load` across consumers via a single helper; the affected tests pass unchanged.
- [ ] `Makefile` (`test-race` target) and `.github/workflows/go.yml` (test job) both pass `-parallel 8` to `go test -race`.
- [ ] `CLAUDE.md` gains a `## Test discipline` section (or equivalent) recording the convention; a contributor reading it can write a new test file in the right shape without prior knowledge.
- [ ] A policy test under `internal/policies/` asserts every `internal/*` test-bearing package has a `setup_test.go` with a `TestMain` declaration; CI fails any future package that omits it.
- [ ] G-0097 promoted to `addressed` via `aiwf promote`; closing commit cites this epic.

## Design decisions (locked at planning time)

| Decision | Rationale |
|---|---|
| **GIT identity is set process-wide via `os.Setenv` in `TestMain`, not per-test via `t.Setenv`.** | `t.Setenv` panics under `t.Parallel`. The values are immutable constants for the test binary's lifetime; `os.Setenv` is the correct primitive. The one test that *deliberately* clears these to provoke a commit failure (`TestApply_RollsBackOnCommitFailure`) keeps its `t.Setenv` and stays serial — `t.Setenv` auto-restores via `t.Cleanup`, so parallel siblings see the restored values when they run after the serial batch. |
| **`-parallel 8` is the cap in CI and local `make test-race`.** | Reliable across the spike's macOS runs; matches typical CI runner core counts. Higher caps risk the macOS git-fan-out flake; lower caps leave performance on the table. Documented in CLAUDE.md alongside the convention. |
| **The shared-tree pattern in `internal/policies/` uses `sync.Once`, not `TestMain`.** | `TestMain` is taken by the env setup; a separate `sync.Once`-guarded loader composes cleanly without conflicting. The `*Tree` is read-only across consumers; reviewer enforces "do not mutate" via a comment at the loader site. |
| **Per-package `TestMain` files are named `setup_test.go`.** | Convention; the spike used the same name. A consistent filename means contributors / AI agents searching for the env-setup site know where to look. |
| **The convention's chokepoint is a `setup_test.go`-presence test, not a per-function `t.Parallel()` audit.** | A presence check is mechanical and cheap; a per-function audit needs an opt-out grammar (filename convention, directive comments) that creates friction without proportional value. Presence of `setup_test.go` is a reasonable proxy: a package that has the file has been audited; the rule lives in CLAUDE.md for the rest. |
| **M-A relaxes "one commit per milestone" to one commit per per-package conversion plus a leading cap-change commit.** | The milestone is refactor-shaped: ~12 packages converted by the same mechanical recipe. A single 50-file commit has poor reviewer signal; per-package commits keep `-race -count=10` regression-bisectable to the specific package that broke. The atomicity rule is about mutation-verb commits with kernel-trailers; refactor milestones can compose multiple commits cleanly. M-B and M-C stay single-commit. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A converted test starts passing under parallelism for a different reason than its assertion claims (race-condition-shape false-positive). | Medium | The conversion is mechanical and adds nothing semantically. Per-package re-run with `-race -count=10` after conversion catches data-race introductions. CI's `flake-hunt` workflow (`go test -race -count=10`) is the standing chokepoint. |
| A package contains tests that touch shared mutable state (package-level vars, global registries) that the inventory missed. | Medium | First wave audits each package's test files for `t.Setenv`, `t.Chdir`, package-level mutation, and global registry use before mass-applying the pattern. The audit's output is a per-package skip-list of tests that stay serial; reviewer-visible. |
| The macOS `git add` segfault recurs at `-parallel 8` on a slower-disk runner. | Low | Cap is conservative; if it recurs, lower to `-parallel 4`. The cap lives in two files (Makefile + workflow); changing it is one-line per file. |
| `cmd/aiwf/` tests that subprocess `aiwf` rely on file-descriptor / process-table limits that parallelism could exhaust. | Low | Per-package audit catches the dense-fan-out files (notably `integration_g37_test.go`) and either caps their parallelism or excludes them from parallel adoption. |

## Milestones

<!-- Bulleted list, ordered by execution sequence. Status lives in each milestone's frontmatter. Recommended order: M-A first (proves the pattern wider than the spike), M-B parallel-safe with M-A, M-C documents the convention so it sticks. -->

- **[M-0091](M-0091-roll-out-testmain-t-parallel-across-internal-test-packages.md)** — Roll out `TestMain` + `t.Parallel` to `internal/*` packages. Apply the spike's pattern across every `internal/*` test package. Per-package audit produces a skip-list (tests that legitimately can't be parallel); the rest gain `t.Parallel()`. `setup_test.go` lands in each package. **First commit** lands the race-cap in `Makefile` + `.github/workflows/go.yml` (+ `flake-hunt.yml`) — the cap is the prerequisite for the parallel adoption that follows, so it must precede the per-package work. **Subsequent commits** convert one package each (relaxing the "one commit per milestone" rule for this refactor-shaped milestone — see Constraints). Memoizing the live-repo `tree.Load(repoRoot)` in `internal/policies` ships in this milestone too, on the same per-package cadence. **Reference for the convention until M-0093 lands: `internal/verb/setup_test.go`** (the spike). · `tdd: none` · depends on: —
- **[M-0092](M-0092-roll-out-testmain-t-parallel-no-ldflags-dedup-to-cmd-aiwf.md)** — Roll out `TestMain` + `t.Parallel` + no-ldflags dedup to `cmd/aiwf/`. Per-file audit (some tests need subprocess isolation; some don't); convert the safe ones; share the no-ldflags binary build via `sync.Once` in `binary_integration_test.go`. Single commit. · `tdd: none` · depends on: M-0091
- **[M-0093](M-0093-document-test-discipline-convention-and-lock-its-chokepoint.md)** — Document the convention and lock its chokepoint. Add a new `## Test discipline` section to `CLAUDE.md` (TestMain for env, t.Parallel default-on, sync.Once for shared expensive fixtures, race-parallel cap). Ship the `setup_test.go`-presence policy test under `internal/policies/` in the same commit so the rule and its enforcement land together. A new test file written under this rule reads as obviously-conformant; a deviation reads as obviously-deviant and fails CI. Single commit. · `tdd: none` · depends on: M-0091, M-0092

(The dependencies are loose: M-0092 can start once M-0091's pattern is established without waiting for M-0091 to fully wrap. M-0093 captures lessons learned from M-0091 and M-0092 and waits for both. If a fourth wave is needed for the deferred `htmlrender` fixture snapshot or the `aiwf init` skeleton snapshot, it spawns its own milestone — this epic accepts the addition rather than letting any milestone bloat.)

## ADRs produced (optional)

(None expected. The convention is documented in CLAUDE.md, not durable enough for ADR shape.)

## Dependencies

- **No upstream blockers.** The conversion touches test files only; no production code, no FSM changes, no ADRs needed first.
- **Compatible with the existing `flake-hunt` workflow.** That workflow runs `go test -race -count=10`; the parallelism cap applies there too (the workflow file change in M-A's first commit covers `go.yml`, `flake-hunt.yml`, and the Makefile in one shot).

## References

- [G-0097](../../gaps/G-0097-test-suite-wall-time-dominated-by-serial-execution-and-per-test-fixture-setup-internal-verb-spike-shows-4-headroom.md) — gap framing, evidence, spike numbers.
- Spike branch `spike/test-parallel` — pattern source; `/tmp/aiwf-test-spike` (worktree).
- [`CLAUDE.md`](../../../CLAUDE.md) — current test-discipline rules; M-C extends the *Go conventions* section.
- [`Makefile`](../../../Makefile) — `test`, `test-race`, `coverage`, `ci` targets; M-A modifies `test-race` (and propagates the cap to `coverage` and `ci` as a single change).
- [`.github/workflows/go.yml`](../../../.github/workflows/go.yml) — CI test job; M-A modifies the `go test -race` line.
- [`.github/workflows/flake-hunt.yml`](../../../.github/workflows/flake-hunt.yml) — race-flake hunter; M-A modifies the `go test -race -count=10` line to add the cap.
- `internal/verb/setup_test.go` (spike) — reference implementation for per-package `TestMain`.
- `cmd/aiwf/integration_test.go::aiwfBinary` — `sync.Once` precedent for M-B.
