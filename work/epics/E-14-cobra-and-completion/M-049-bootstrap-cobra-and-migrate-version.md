---
id: M-049
title: Bootstrap Cobra and migrate version
status: in_progress
parent: E-14
acs:
    - id: AC-1
      title: Cobra dependency added to go.mod with one-line justification in commit message
      status: met
    - id: AC-2
      title: Cobra root command and subcommand routing structure in cmd/aiwf
      status: met
    - id: AC-3
      title: version verb migrated; --format=json envelope shape preserved byte-exact
      status: met
    - id: AC-4
      title: Exit codes 0/1/2/3 preserved end-to-end through Cobra dispatch
      status: met
    - id: AC-5
      title: Subprocess integration test for version verb passes against the migrated binary
      status: met
    - id: AC-6
      title: Auto-completion design principle added to CLAUDE.md
      status: met
---

## Goal

Add Cobra to the module, set up the root command and subcommand routing scaffold, and migrate the simplest existing verb (`version`) to validate the pattern end-to-end. Also locks in the auto-completion design principle in `CLAUDE.md` so subsequent migration milestones operate under it as a guiding constraint.

## Approach

`version` is the right migration test bed because it already has a subprocess integration test (per CLAUDE.md "test the seam") and exercises both `--format=json` envelope handling and the exit-code contract. Once the pattern works end-to-end on `version`, the remaining verbs follow mechanically. This milestone is intentionally narrow — its job is to prove the shape, not to migrate the surface.

## Acceptance criteria

### AC-1 — Cobra dependency added to go.mod with one-line justification in commit message

### AC-2 — Cobra root command and subcommand routing structure in cmd/aiwf

### AC-3 — version verb migrated; --format=json envelope shape preserved byte-exact

### AC-4 — Exit codes 0/1/2/3 preserved end-to-end through Cobra dispatch

### AC-5 — Subprocess integration test for version verb passes against the migrated binary

### AC-6 — Auto-completion design principle added to CLAUDE.md

