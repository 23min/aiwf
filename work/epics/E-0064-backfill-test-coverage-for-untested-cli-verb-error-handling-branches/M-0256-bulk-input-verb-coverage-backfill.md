---
id: M-0256
title: Bulk-input verb coverage backfill
status: draft
parent: E-0064
depends_on:
    - M-0252
tdd: required
acs:
    - id: AC-1
      title: Every bulk-input verb group branch tested or ignored
      status: open
      tdd_phase: red
    - id: AC-2
      title: Scoped coverage-gate reports zero findings
      status: open
      tdd_phase: red
---

## Goal

Clear every branch `branch-coverage-audit` currently flags in the
bulk-input verb group — `importcmd`, `render`, `check`+`check/provenance`
— using the shared failure fixtures M-0252 builds.

## Context

M-0252 lands the reusable fixtures for the failure modes these guards
share. This group's verbs read larger, more structurally varied input
(imported entity files, render targets, the full tree under `aiwf check`)
than the other three consumer groups, so a couple of its flagged sites may
need a fixture beyond M-0252's five shared ones — a malformed import
source, or a render-target-specific failure — surfaced during
implementation rather than pre-designed here.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0256 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — Every branch `branch-coverage-audit` flags (base =
  the commit before M-0238/AC-3's rename) within `internal/cli/{importcmd,
  render,check}` (including `check/provenance.go`) carries either a
  passing test (reusing M-0252's fixtures where the failure mode matches,
  or a new fixture where it doesn't) or a `//coverage:ignore <reason>`.
- **AC-2 candidate** — `make coverage-gate`, run with `AIWF_COVERAGE_BASE`
  set to the pre-M-0238 commit, reports zero findings for the files listed
  in AC-1.

### AC-1 — Every bulk-input verb group branch tested or ignored

### AC-2 — Scoped coverage-gate reports zero findings

## Constraints

- Reuse M-0252's fixtures for shared failure modes rather than
  reimplementing them per file; a genuinely new fixture (e.g. malformed
  import source) is scoped to this milestone only.
- Per-site judgment only: real test where triggerable, honest
  `//coverage:ignore <reason>` otherwise.

## Out of scope

- Entity-lifecycle, contract, diagnostic, and non-CLI infra files —
  M-0252, M-0253, and M-0255's job.
- Any change to error-handling behavior beyond what's needed to make a
  branch testable.

## Dependencies

- M-0252 — its shared fixtures must exist before this milestone starts.

## References

- **E-0064** — parent epic.
- **M-0252** — shared fixtures this milestone consumes.
