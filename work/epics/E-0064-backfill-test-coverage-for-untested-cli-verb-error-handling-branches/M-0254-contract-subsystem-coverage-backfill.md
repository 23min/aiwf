---
id: M-0254
title: Contract subsystem coverage backfill
status: draft
parent: E-0064
depends_on:
    - M-0252
tdd: required
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the contract
subsystem — `contract/recipes`, `contract/verify`, `contract/bind`,
`contract/unbind` — using the shared failure fixtures M-0252 builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share. `contract/recipes.go` alone carries the single largest
concentration of flagged sites of any file in the epic's scope, so this
milestone is sized to that file plus its three siblings in the same
package.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0254 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — Every branch `branch-coverage-audit` flags (base =
  the commit before M-0238/AC-3's rename) within `internal/cli/contract/
  {recipes,verify,bind,unbind}.go` carries either a passing test (reusing
  M-0252's fixtures where the failure mode matches) or a
  `//coverage:ignore <reason>`.
- **AC-2 candidate** — `make coverage-gate`, run with `AIWF_COVERAGE_BASE`
  set to the pre-M-0238 commit, reports zero findings for the files listed
  in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Entity-lifecycle, diagnostic, bulk-input, and non-CLI infra files —
  M-0252, M-0253, M-0255, and M-0256's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.
