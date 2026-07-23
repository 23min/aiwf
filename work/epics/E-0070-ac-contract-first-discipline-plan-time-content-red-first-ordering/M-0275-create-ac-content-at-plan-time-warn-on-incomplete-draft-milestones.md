---
id: M-0275
title: Create AC content at plan time; warn on incomplete draft milestones
status: draft
parent: E-0070
tdd: required
acs:
    - id: AC-1
      title: Draft milestone with zero ACs raises a warning-severity finding
      status: open
    - id: AC-2
      title: Draft milestone with an empty AC body raises the finding
      status: open
    - id: AC-3
      title: The draft-AC finding is archive-scoped (silent on archived milestones)
      status: open
    - id: AC-4
      title: aiwfx-plan-milestones adds and body-fills ACs before its merge-to-main step
      status: open
    - id: AC-5
      title: aiwfx-start-milestone preflight reframes ACs as expected-to-pre-exist
      status: open
---

# M-0275 — Create AC content at plan time; warn on incomplete draft milestones

## Goal

Close G-0440: move AC-entity creation and body-filling into
`aiwfx-plan-milestones` (before its merge-to-main step) so milestones don't
land on main with empty or missing ACs, and add a warning-severity check-time
finding that surfaces a `draft` milestone with zero ACs or empty AC bodies.

## Context

`aiwfx-plan-milestones` merges planning to main without ever calling
`aiwf add ac` — that happens later, inside `aiwfx-start-milestone`'s preflight,
one FSM stage after the milestone is already visible on main. A reader without
the epic's worktree checked out then can't see what the milestone requires.
G-0216/D-0039 already built the "contract before code" guard, but it fires at
`draft -> in_progress` — one stage after this visibility gap. This milestone
extends that family one stage earlier, as a warning (not a block), because
`draft` is a legitimate mid-planning state. Full rationale in D-0047.

## Acceptance criteria

<!-- Prose shape; formalized via `aiwf add ac` at aiwfx-start-milestone.
     Each is observable behavior with a mechanical assertion. -->

1. A new warning-severity check-time finding fires for a non-archived `draft`
   milestone with zero AC entities — check-level test.
2. The finding (or a paired one) fires for a non-archived `draft` milestone
   with any AC whose body subsection is empty — check-level test.
3. The finding is archive-scoped via the existing `entity.IsArchivedPath`
   guard — an archived `draft` milestone does not fire — check-level test.
4. `aiwfx-plan-milestones` calls `aiwf add ac` and fills each AC body before
   its merge-to-main step — structural policy test asserting the call appears
   in the plan step and precedes the merge step (skill-edit backstop).
5. `aiwfx-start-milestone`'s preflight reframes ACs as expected-to-pre-exist,
   retaining the "add them now" fallback as a recovery path for hand-written
   specs — structural policy test.

### AC-1 — Draft milestone with zero ACs raises a warning-severity finding

### AC-2 — Draft milestone with an empty AC body raises the finding

### AC-3 — The draft-AC finding is archive-scoped (silent on archived milestones)

### AC-4 — aiwfx-plan-milestones adds and body-fills ACs before its merge-to-main step

### AC-5 — aiwfx-start-milestone preflight reframes ACs as expected-to-pre-exist

## Constraints

- Warn, never block, at `draft` — mirrors D-0039's block-at-transition /
  warn-at-rest split; no change to the existing `draft -> in_progress` block.
- The new finding reuses the file's existing archive-scoping convention — no
  new grandfather/timestamp mechanism.

## Design notes

- Implements D-0047 point 2; closes G-0440.
- Extends the `internal/check/acs.go` finding family alongside
  `milestoneDoneIncompleteACs`.

## Surfaces touched

- `internal/check/acs.go` (new warning finding)
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`
- `internal/policies/` (structural tests for the skill edits)

## Out of scope

- The seeding fix (M-0274) and the ordering gate (M-0276).
- Blocking (as opposed to warning) an incomplete `draft` milestone.

## Dependencies

- None (independent of M-0274 and M-0276). Sequenced after M-0274 by soft
  preference so plan-time AC creation, once it is the practice, produces
  ACs born correctly at `""`.

## References

- G-0440 — the gap this milestone closes.
- D-0047 — Contract-first AC timing and red-first ordering enforcement.
- G-0216 / D-0039 — the AC-completeness guard precedent this extends.
