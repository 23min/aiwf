---
id: M-0252
title: Shared CLI-verb failure fixtures and non-CLI infra coverage backfill
status: draft
parent: E-0064
tdd: required
acs:
    - id: AC-1
      title: Reusable test fixtures exist for each shared CLI-verb failure mode
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

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0252 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — Reusable test fixtures exist (e.g. under
  `internal/cliutil/testutil` or a sibling package) for each shared failure
  mode: root-resolution failure, actor-resolution failure, repo-lock
  contention (`cliutil.AcquireRepoLock`), malformed/corrupt tree
  (`tree.Load`), and a bad `--format` flag. Each fixture is documented with
  the exact condition it simulates.
- **AC-2 candidate** — Every branch `branch-coverage-audit` flags (base =
  the commit before M-0238/AC-3's rename) within `internal/verb/*`,
  `internal/gitops/refs.go`, `internal/stresstest/*`, `internal/check/*`,
  `internal/cellcoverage/*`, and `internal/cli/cliutil/*` carries either a
  passing test built on these fixtures or a `//coverage:ignore <reason>`
  naming why the branch isn't triggerable, mirroring
  `internal/cli/archive/archive.go:120`.
- **AC-3 candidate** — `make coverage-gate`, run with `AIWF_COVERAGE_BASE`
  set to the pre-M-0238 commit, reports zero findings for the files listed
  in AC-2.

### AC-1 — Reusable test fixtures exist for each shared CLI-verb failure mode

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
