---
id: E-0064
title: Backfill test coverage for untested CLI verb error-handling branches
status: proposed
---

## Goal

Every currently-flagged untested CLI-verb error-handling branch gets either a
real regression test or a documented `//coverage:ignore`, so
`make coverage-gate` reports zero findings against these sites and the
diff-scoped coverage gate stops firing on incidental future touches to this
code.

## Context

M-0238/AC-3 mechanically rewrote every bare `fmt.Fprintf(os.Stderr, ...)` /
`fmt.Println(...)` call site under `internal/cli/` to route through the
`cliutil` text-output wrapper set — same stream, same bytes, no logic
change. The rename touched lines inside pre-existing, never-exercised
`if err != nil { <print>; return ... }` guards (typically wrapping
`cliutil.ResolveRoot`, `cliutil.ResolveActor`, `cliutil.AcquireRepoLock`,
`tree.Load`, or a verb-specific validation step), and the diff-scoped
`branch-coverage-audit` policy flagged every one of them — not because the
rename broke anything, but because it was the first change to touch those
lines since the coverage gate started scoping against `origin/main`.

That surfaced G-0386: a large fraction of this codebase's CLI-verb
infrastructure-failure paths have no test asserting they print the right
message and return the right exit code. The gap is real independent of
M-0238 or E-0061 (already merged to `main`) — re-running
`branch-coverage-audit` against the pre-M-0238 base and current `main`
confirms 188 of the sites are still live today, across 39 files.

This repo already has precedent for the correct per-site judgment call at
`internal/cli/archive/archive.go:120` (`//coverage:ignore
cliutil.ResolveRoot only fails on missing aiwf.yaml + non-existent --root
path`), but most of the flagged files never went through that exercise.

## Scope

### In scope

- Every line flagged by `branch-coverage-audit` when run with
  `AIWF_COVERAGE_BASE` set to the commit before M-0238/AC-3's rename
  (`2ac84846^`) against current `main` — 188 findings across 39 files,
  spanning `internal/cli/*` verb packages, `internal/verb/`,
  `internal/gitops/refs.go`, `internal/stresstest/`, `internal/check/`, and
  `internal/cellcoverage/`.
- Per flagged site: a real test where the failure condition is genuinely
  triggerable (a malformed entity file, a bad `--format` flag, simulated
  lock contention, a corrupt tree), or an honest `//coverage:ignore`
  naming why it isn't — mirroring the `archive.go:120` precedent.
- Shared test helpers where multiple verbs guard the same failure shape
  (e.g. a fixture that reliably fails `cliutil.AcquireRepoLock`), so the
  fix doesn't reinvent triggering machinery per file.

### Out of scope

- Any refactor to these files beyond what's needed to make a branch
  testable (no unrelated cleanup, no signature changes).
- Coverage improvements beyond the flagged lines — this epic closes
  G-0386's named debt, not a general coverage-raising initiative.
- Changing error-handling *behavior* (message wording, exit codes) except
  where a change is required to make a branch deterministically
  triggerable.
- Coordination with any other in-flight epic — this branches from `main`
  directly and is independent of E-0061 or any successor.

## Constraints

- Per-site judgment only, never a blanket suppression: a real test where
  triggerable, an honest `//coverage:ignore <reason>` otherwise. A
  milestone that lands `//coverage:ignore` annotations without an
  accompanying "why not testable" rationale per line fails review.
- `//coverage:ignore` rationale style follows the existing
  `archive.go:120` precedent — one line, names the specific condition that
  makes the branch untestable.
- Branched from `main` directly, not from any epic branch.

## Success criteria

- [ ] `make coverage-gate`, run with `AIWF_COVERAGE_BASE` set to the
      pre-M-0238 commit (`2ac84846^`) against current `main`, reports zero
      findings.
- [ ] Every site in that comparison carries either a passing test that
      exercises it or a `//coverage:ignore` with a rationale line.
- [ ] The standard diff-scoped `make coverage-gate` (base =
      `origin/main`) stays clean through the epic's own commits.

## Open questions

None — resolved during `aiwfx-plan-milestones`: milestones split by shared
guard shape, with a foundational milestone building reusable failure
fixtures that the four verb-family milestones consume in parallel.

## Milestones

- `M-0252` — Shared CLI-verb failure fixtures and non-CLI infra coverage
  backfill (`internal/verb/*`, `internal/gitops`, `internal/stresstest`,
  `internal/check`, `internal/cellcoverage`, `cliutil/*`) · depends on: —
- `M-0253` — Entity-lifecycle verb coverage backfill (`add`, `promote`,
  `retitle`, `rename`, `reallocate`, `cancel`, `milestone`, `update`,
  `rewidth`, `editbody`) · depends on: `M-0252`
- `M-0254` — Contract subsystem coverage backfill (`contract/recipes`,
  `contract/verify`, `contract/bind`, `contract/unbind`) · depends on:
  `M-0252`
- `M-0255` — Diagnostic and introspection verb coverage backfill
  (`doctor`+`selfcheck`, `status`, `show`, `history`, `list`, `whoami`,
  `schema`, `template`) · depends on: `M-0252`
- `M-0256` — Bulk-input verb coverage backfill (`importcmd`, `render`,
  `check`+`check/provenance`) · depends on: `M-0252`

## References

- G-0386 — Backfill test coverage for ~194 untested CLI verb
  error-handling branches (the gap this epic closes).
- `internal/cli/archive/archive.go:120` — precedent
  `//coverage:ignore` rationale style.
- `internal/policies/branch_coverage_audit.go` — the diff-scoped
  coverage-gate policy that surfaces these findings.
