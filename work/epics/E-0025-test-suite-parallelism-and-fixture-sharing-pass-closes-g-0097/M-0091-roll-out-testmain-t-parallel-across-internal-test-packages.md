---
id: M-0091
title: Roll out TestMain + t.Parallel across internal/* test packages
status: draft
parent: E-0025
tdd: none
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
