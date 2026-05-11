---
id: M-0094
title: Add aiwf check finding epic-active-no-drafted-milestones
status: draft
parent: E-0028
tdd: required
---

# M-0094 — Add `aiwf check` finding `epic-active-no-drafted-milestones`

## Goal

Add a new `aiwf check` warning finding `epic-active-no-drafted-milestones` that fires when an epic at status `active` has zero milestones at status `draft`. The finding informs the `aiwfx-start-epic` skill's preflight: an epic is not ready for activation until at least one milestone is drafted.

## Context

E-0028's start-epic ritual needs a kernel-checkable signal for "is this epic actually ready to activate?" The body-completeness check (`entity-body-empty`, M-0066) already covers Goal/Scope/Out-of-scope prose. The remaining preflight gap per G-0063 is the drafted-milestone check — that an epic going active has work queued, not just text on a page.

This is a standalone kernel rule. It does not depend on M-0095 (sovereign-act enforcement); the two land in parallel and the skill in M-0096 consumes both.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- One new finding code: `epic-active-no-drafted-milestones`, severity warning.
- Trigger: any epic with `status: active` whose child milestone set contains zero entries at `status: draft`.
- Hint text points at G-0063's framing and the skill's preflight role.
- Implementation lives in `internal/check/` alongside the existing kind-scoped rules; tests under `internal/check/<name>_test.go` follow the fixture-tree convention used by `entity_body_test.go`.

## Dependencies

- None. Standalone kernel rule; lands in parallel with M-0095.

## References

- E-0028 epic spec.
- G-0063 — preflight checks table, row 2.
- `internal/check/entity_body.go` and `_test.go` — reference shape for new check rules.
