---
id: M-0252
title: Shared CLI-verb failure fixtures and non-CLI infra coverage backfill
status: in_progress
parent: E-0064
tdd: required
acs:
    - id: AC-1
      title: Reusable test fixtures exist for each shared CLI-verb failure mode
      status: open
      tdd_phase: green
    - id: AC-2
      title: Non-CLI-infra flagged branches are tested or documented
      status: open
      tdd_phase: red
    - id: AC-3
      title: Coverage gate is clean for the non-CLI-infra group
      status: open
      tdd_phase: red
---

## Goal

Establish reusable test fixtures for the handful of failure modes shared
across CLI-verb error-handling guards, and use them to clear every branch
`branch-coverage-audit` currently flags in `internal/verb/*`,
`internal/gitops/refs.go`, `internal/stresstest/*`, `internal/check/*`,
`internal/cellcoverage/*`, and `internal/cli/cliutil/*` itself.

## Context

E-0064 closes G-0386's test-coverage debt: ~194 (188 currently live)
CLI-verb error-handling branches surfaced as untested when M-0238/AC-3's
mechanical print-call rename touched lines inside guards nothing had ever
exercised. This is the foundational milestone — it builds the shared
triggering fixtures (root-resolution failure, actor-resolution failure,
repo-lock contention, malformed/corrupt tree, bad `--format` flag) that
M-0253 through M-0256 (parallel, all depending on this milestone) will
reuse, and proves them against real call sites by applying them to the
smallest flagged group first.

## Acceptance criteria

### AC-1 — Reusable test fixtures exist for each shared CLI-verb failure mode

Reusable test fixtures exist (e.g. under `internal/cliutil/testutil` or a
sibling package) for each shared failure mode: root-resolution failure,
actor-resolution failure, repo-lock contention
(`cliutil.AcquireRepoLock`), malformed/corrupt tree (`tree.Load`), and a
bad `--format` flag. Each fixture is documented with the exact condition
it simulates, so M-0253 through M-0256 can reuse it without re-deriving
the trigger.

### AC-2 — Non-CLI-infra flagged branches are tested or documented

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename) within `internal/verb/*`, `internal/gitops/refs.go`,
`internal/stresstest/*`, `internal/check/*`, `internal/cellcoverage/*`,
and `internal/cli/cliutil/*` carries either a passing test built on the
AC-1 fixtures or a `//coverage:ignore <reason>` naming why the branch
isn't triggerable, mirroring `internal/cli/archive/archive.go:120`.

### AC-3 — Coverage gate is clean for the non-CLI-infra group

`make coverage-gate`, run with `AIWF_COVERAGE_BASE` set to the pre-M-0238
commit, reports zero findings for the files listed in AC-2.

## Constraints

- Per-site judgment only: a real test where the failure is genuinely
  triggerable, an honest `//coverage:ignore <reason>` otherwise — never a
  blanket suppression.
- Fixtures are built generic enough for M-0253–M-0256 to reuse as-is; a
  fixture that only works for one call site defeats the point of doing
  this milestone first.

## Out of scope

- Entity-lifecycle, contract, diagnostic, and bulk-input verb files —
  M-0253 through M-0256's job.
- Any change to error-handling behavior (message wording, exit codes)
  beyond what's needed to make a branch deterministically triggerable.

## Dependencies

- E-0064 epic spec (committed).
- No prior milestone — this is first.

## References

- **E-0064** — parent epic.
- **G-0386** — the gap this epic (and this milestone) closes.
- `internal/cli/archive/archive.go:120` — precedent `//coverage:ignore`
  rationale style.
- `internal/policies/branch_coverage_audit.go` — the diff-scoped
  coverage-gate policy that surfaces these findings.
