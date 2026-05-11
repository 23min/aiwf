---
id: M-0105
title: aiwfx-start-milestone sequencing alignment
status: draft
parent: E-0030
depends_on:
    - M-0102
    - M-0103
tdd: required
---

## Goal

Align `aiwfx-start-milestone`'s step order with M-0104's epic-side fix: `aiwf promote M-NNN draft → in_progress` lands on the parent epic branch, then the milestone work branch is cut via the new `aiwf authorize --branch milestone/M-NNN-<slug>` surface.

## Context

M-0104 establishes the pattern for `aiwfx-start-epic`; this milestone applies the same shape one level down at the milestone-start ritual. [ADR-0010](../../../docs/adr/ADR-0010-branch-model-ritualized-work-on-branches-author-iteration-on-main.md)'s symmetric rule for milestones: the promote-to-in_progress is a state-announcement that belongs on the parent epic branch (which already exists at this point), not on the milestone work branch (which hasn't been cut yet).

Cross-repo via the fixture pattern, same shape as M-0104. The canonical authoring location is `internal/policies/testdata/aiwfx-start-milestone/SKILL.md` in this repo.

## Out of scope

- `aiwfx-start-epic` (M-0104, sibling).
- Kernel finding (M-0106).
- AC-level branch behavior — ACs ride on the milestone branch alongside test/code commits per ADR-0010; no separate AC-branch convention is in scope here.

## Dependencies

- **M-0102** — `--branch` flag.
- **M-0103** — preflight refuses dispatch without ritual branch context.

## Open questions for AC drafting

- **Inheritance of authorize scope:** Does the milestone's `aiwf authorize` open a *new* scope (nested under the epic's), or extend the epic's existing scope? Likely new sub-scope per ADR-0009's substrate/driver split, but explicit decision goes here.
- **Parent branch detection:** The ritual needs to know which epic branch to land the promote on. Read `aiwf show M-NNN` for the parent, then check that branch is currently checked out? Or require the operator to be on the epic branch already?

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0105` time. -->
