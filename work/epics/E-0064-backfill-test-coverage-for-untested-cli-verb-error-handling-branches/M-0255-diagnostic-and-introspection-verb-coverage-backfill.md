---
id: M-0255
title: Diagnostic and introspection verb coverage backfill
status: in_progress
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Every diagnostic/introspection-group branch tested or ignored
      status: met
      tdd_phase: done
    - id: AC-2
      title: Scoped coverage-gate reports zero findings
      status: open
      tdd_phase: red
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the
diagnostic/introspection verb group — `doctor`+`selfcheck`, `status`,
`show`, `history`, `list`, `whoami`, `schema`, `template` — plus
`archive` and `authorize`, using the shared failure fixtures M-0252
builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share. `doctor/selfcheck.go` carries a large concentration of flagged
sites on its own; the rest of this group is read-oriented verbs with a
thinner failure surface each. `archive` and `authorize` don't fit this
milestone's read-oriented theme — they're mutating verbs — but neither
was assigned to any of E-0064's other four milestones; folding their 10
remaining flagged lines in here (rather than a sixth milestone) keeps
the epic's "zero findings" success criterion reachable without adding a
milestone for two files.

## Acceptance criteria

### AC-1 — Every diagnostic/introspection-group branch tested or ignored

Every branch `branch-coverage-audit` flags (base = the commit before
M-0238/AC-3's rename, `2ac84846^`) within
`internal/cli/{doctor,status,show,history,list,whoami,schema,template,
archive,authorize}` carries either a passing test (reusing M-0252's
fixtures where the failure mode matches) or a
`//coverage:ignore <reason>`.

### AC-2 — Scoped coverage-gate reports zero findings

`make coverage-gate`'s underlying policy test, run with `AIWF_COVERAGE_BASE`
set to the pre-M-0238 commit, reports zero findings for the files listed
in AC-1.

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Entity-lifecycle, contract, bulk-input, and non-CLI infra files —
  M-0252, M-0253, M-0254, and M-0256's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.
