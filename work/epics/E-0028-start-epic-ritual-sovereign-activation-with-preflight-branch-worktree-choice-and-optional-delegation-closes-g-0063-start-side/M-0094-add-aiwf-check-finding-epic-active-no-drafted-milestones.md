---
id: M-0094
title: Add aiwf check finding epic-active-no-drafted-milestones
status: in_progress
parent: E-0028
tdd: required
acs:
    - id: AC-1
      title: rule fires warning when active epic has zero drafted milestones
      status: open
      tdd_phase: done
    - id: AC-2
      title: rule does not fire when active epic has at least one drafted milestone
      status: open
      tdd_phase: red
    - id: AC-3
      title: rule does not fire when epic is at status proposed, done, or cancelled
      status: open
      tdd_phase: red
    - id: AC-4
      title: finding hint text references the start-epic preflight role and G-0063
      status: open
      tdd_phase: red
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

### AC-1 — rule fires warning when active epic has zero drafted milestones

A fixture tree under `internal/check/testdata/` with an epic at status `active` and zero milestones at status `draft` (zero milestones at all is the simplest positive shape; "zero drafts among existing milestones" is the same case under reading A). Driving the test through `check.Run` (per CLAUDE.md *Test the seam, not just the layer*) produces a `findings.Finding` with code `epic-active-no-drafted-milestones`, severity `warning`, target the epic's id.

### AC-2 — rule does not fire when active epic has at least one drafted milestone

A sibling fixture with the same `active` epic plus ≥1 milestone at status `draft`. Driving through `check.Run`, the result set contains no finding with code `epic-active-no-drafted-milestones`. Covers the negative branch — drafts present, no warning.

### AC-3 — rule does not fire when epic is at status proposed, done, or cancelled

Table-driven test across each non-`active` epic status — `proposed`, `done`, `cancelled` — each with zero drafted milestones (the firing condition under reading A). For every status, the result set contains no finding with code `epic-active-no-drafted-milestones`. Covers the kind/status guard; ensures the rule is scoped exactly to `active` epics, not to all epics-with-no-drafts.

### AC-4 — finding hint text references the start-epic preflight role and G-0063

The finding's hint surface mentions G-0063 (gap framing) and the start-epic preflight role (so a reader who lands on the finding via `aiwf check` can navigate to the framing without re-deriving it). Structural assertion on the hint string (substring grep is acceptable here per CLAUDE.md *Substring assertions are not structural assertions* — the hint surface is a single short string, not a structured document where placement matters).

