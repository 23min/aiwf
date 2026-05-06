---
id: M-049
title: Bootstrap Cobra and migrate version
status: draft
parent: E-14
acs:
    - id: AC-1
      title: Cobra dependency added to go.mod with one-line justification in commit message
      status: open
---

## Goal

Add Cobra to the module, set up the root command and subcommand routing scaffold, and migrate the simplest existing verb (`version`) to validate the pattern end-to-end. Also locks in the auto-completion design principle in `CLAUDE.md` so subsequent migration milestones operate under it as a guiding constraint.

## Approach

`version` is the right migration test bed because it already has a subprocess integration test (per CLAUDE.md "test the seam") and exercises both `--format=json` envelope handling and the exit-code contract. Once the pattern works end-to-end on `version`, the remaining verbs follow mechanically. This milestone is intentionally narrow — its job is to prove the shape, not to migrate the surface.

## Acceptance criteria

### AC-1 — Cobra dependency added to go.mod with one-line justification in commit message

