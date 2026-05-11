---
id: M-0091
title: Roll out TestMain + t.Parallel across internal/* test packages
status: draft
parent: E-0025
tdd: none
acs:
    - id: AC-1
      title: race-parallel cap lands in Makefile and workflows in a leading commit
      status: open
    - id: AC-2
      title: every internal/* test-bearing package has setup_test.go with TestMain
      status: open
    - id: AC-3
      title: t.Parallel adopted on every test that does not need serial execution
      status: open
    - id: AC-4
      title: internal/policies shares the live-repo tree.Load via a sync.Once helper
      status: open
    - id: AC-5
      title: go test -race -parallel 8 ./internal/... reliable across 10 runs
      status: open
    - id: AC-6
      title: wall-time baseline and post-conversion numbers recorded at wrap
      status: open
---

# M-0091 — Roll out TestMain + t.Parallel across internal/* test packages

## Goal

Apply the spike's `TestMain` + `t.Parallel()` pattern across every `internal/*` test package, cap `-race` parallelism at 8 in `Makefile` and the GitHub workflows, and memoize the live-repo `tree.Load(repoRoot)` in `internal/policies/`. After this milestone, `go test ./internal/...` is meaningfully faster at default parallelism and `go test -race -parallel 8 ./internal/...` is reliable across 10 consecutive runs.

## Context

The spike branch `spike/test-parallel` proved the pattern on `internal/verb/` (~4× faster non-race, ~2.4× with race). G-0097 catalogues the rest of `internal/*` as following the same shape — every test package re-does its own git-init / tree-load and almost none call `t.Parallel()`. The pattern is mechanical; the work is rolling it out package-by-package and recording the skip-list per package for tests that legitimately must stay serial.

Per the epic's Constraints, this milestone explicitly **relaxes "one commit per milestone"** to one commit per per-package conversion plus a leading cap-change commit. The cap-change must precede the conversions because `-parallel 8` is the safety net that lets the per-package commits run under `-race` without flakes.

## Acceptance criteria

(ACs allocated via `aiwf add ac`; bodies follow below.)

## Constraints

- **No test semantics change.** The conversion is mechanical — adding `t.Parallel()` and moving env setup to `TestMain`. A converted test that newly passes for a different reason than its assertion claims fails review (per CLAUDE.md "Don't paper over a test failure"). The chokepoint is the per-package re-run with `-race -count=10` recorded in the commit body.
- **One commit per per-package conversion plus the leading cap-change commit.** Each commit carries `aiwf-verb`, `aiwf-entity: M-0091`, and `aiwf-actor` trailers. Atomicity by *package*, not by milestone, is deliberate — see the epic's design decisions table.
- **`os.Setenv` for the GIT identity, not `t.Setenv`.** `t.Setenv` panics under `t.Parallel`. The values are immutable for the test binary's lifetime, so `os.Setenv` is the correct primitive. The one exception is `TestApply_RollsBackOnCommitFailure` (and any sibling that deliberately clears the identity to provoke a commit failure), which keeps its `t.Setenv` and stays serial.
- **Forward-compatibility.** The pattern must not introduce package-level mutable state that a future test could surprise itself with. The shared `*Tree` is read-only; the `// do not mutate` comment is reviewer-enforced.

## Design notes

- The conversion pattern is the spike's, verbatim: `setup_test.go` per package, `os.Setenv` block in `TestMain`, `t.Parallel()` first-line on every parallelizable test. No deviation; no per-package cleverness.
- The `-parallel 8` cap is conservative. If a slower-disk runner re-surfaces the macOS git-add segfault at 8, the cap drops to 4 in one-line follow-ups (Makefile + two workflow files).
- Memoizing `tree.Load(repoRoot)` is in this milestone (not M-0092 or M-0093) because the consumers are all under `internal/policies/` — co-locating the change with the rest of the `internal/*` rollout keeps the per-package commit boundaries clean.

## Surfaces touched

- `Makefile` — `test-race`, `coverage`, `ci` targets
- `.github/workflows/go.yml`, `.github/workflows/flake-hunt.yml`
- `internal/*/setup_test.go` (new, one per package)
- `internal/policies/` — shared `*Tree` helper

## Out of scope

- `cmd/aiwf/` test conversions — those land in M-0092.
- The `CLAUDE.md ## Test discipline` section and the `setup_test.go`-presence policy test — those land in M-0093.
- `internal/htmlrender/htmlrender_test.go::writeFixtureTree` snapshot conversion — out per the epic.
- Pre-baked `aiwf init`-ed skeleton tempdir — out per the epic.

## Dependencies

- None. The conversion touches test files only.

## References

- E-0025 epic spec (this milestone's parent) — design decisions, risks, constraints.
- G-0097 — gap framing, evidence, spike numbers.
- Spike branch `spike/test-parallel`; `internal/verb/setup_test.go` — reference implementation.
- CLAUDE.md *Go conventions* — testing rules; this milestone's chokepoint commits comply but do not extend.

## Work log

(filled during implementation)

## Decisions made during implementation

- (none yet)

## Validation

(pasted at wrap: baseline wall time, post-conversion wall time, the 10-run `-race -parallel 8` log)

## Deferrals

- (none yet)

## Reviewer notes

- (none yet)

### AC-1 — race-parallel cap lands in Makefile and workflows in a leading commit

`Makefile`'s `test-race` target gains `-parallel 8`; the `coverage` and `ci` targets propagate the same cap where they invoke race-mode tests. `.github/workflows/go.yml`'s race-mode test step gains the same flag. `.github/workflows/flake-hunt.yml`'s `go test -race -count=10` line gains the same flag. The commit message documents the rationale (race + heavy subprocess fan-out flakes on macOS at default parallelism; CI Linux less affected but uniform cap keeps the chokepoint single-valued). After this commit, all three race-test surfaces agree.

### AC-2 — every internal/* test-bearing package has setup_test.go with TestMain

For each package in the scope list — `internal/verb`, `internal/check`, `internal/initrepo`, `internal/htmlrender`, `internal/policies`, `internal/aiwfyaml`, `internal/contractverify`, `internal/tree`, `internal/gitops`, `internal/skills`, `internal/config`, plus any others discovered during audit — a `setup_test.go` is added with a `TestMain(m *testing.M)` that calls `os.Setenv` once for `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL` before `m.Run()`. The filename is uniform; the structure mirrors `internal/verb/setup_test.go` from the spike. Tests in the package no longer call `t.Setenv` for these four variables (with the documented serial-batch exceptions called out in AC-3).

### AC-3 — t.Parallel adopted on every test that does not need serial execution

For each converted package, the per-package audit produces a written skip-list of tests that stay serial — those that call `t.Setenv`/`t.Chdir`, mutate a package-level var, or guard a global the parallel suite depends on. Every test not on the skip-list calls `t.Parallel()` as its first non-`t.Helper()` statement. The skip-list lives in the package's `setup_test.go` as a comment (`// Serial tests: …`) so a contributor can see at a glance what is intentionally serial and why. The per-package conversion commits each cite the skip-list in their commit body.

### AC-4 — internal/policies shares the live-repo tree.Load via a sync.Once helper

A new helper in `internal/policies/` loads the repo tree once and returns the shared `*Tree` to consumers (`TestPolicy_ThisRepoTreeIsClean`, `TestPolicy_ThisRepoDriftCheckClean`, `internal/policies/m080_test.go::loadM080Spec`, and any other live-repo-tree readers discovered during conversion). The helper carries a `// do not mutate` comment at its definition. `TestMain` and the shared loader compose cleanly (env setup in `TestMain`; tree memoization in a separate `sync.Once`).

### AC-5 — go test -race -parallel 8 ./internal/... reliable across 10 runs

After all per-package commits land, the milestone records a 10-run loop of `go test -race -parallel 8 ./internal/...` with zero flakes and zero timeouts. The run log is pasted into the milestone's Validation section at wrap. If a flake surfaces, it is root-caused (not papered over per CLAUDE.md "Don't paper over a test failure") and the fix lands either as a per-package amendment or as a new gap if it's a real data race the conversion exposed.

### AC-6 — wall-time baseline and post-conversion numbers recorded at wrap

The milestone records `go test ./internal/...` wall time before the first conversion commit and after the last, on the same hardware. The target per the epic is ≥2× faster at default parallelism; the actual numbers go into Validation at wrap. If the target is missed, the milestone documents why in Reviewer notes (not a wrap blocker; the epic's success criteria are the hard bar).

