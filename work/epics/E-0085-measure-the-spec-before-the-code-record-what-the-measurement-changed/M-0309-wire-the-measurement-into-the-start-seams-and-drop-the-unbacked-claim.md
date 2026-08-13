---
id: M-0309
title: Wire the measurement into the start seams and drop the unbacked claim
status: draft
parent: E-0085
depends_on:
    - M-0308
tdd: required
acs:
    - id: AC-1
      title: The milestone preflight names the ritual, not a bare judgment call
      status: open
    - id: AC-2
      title: A gap-closing patch measures the gap's claims before the change
      status: open
    - id: AC-3
      title: No surface promises testable criteria without naming the method
      status: open
---

## Goal

Route every seam that starts work through the measurement ritual, and stop the
one surface that promises testable acceptance criteria it has no method to
produce.

## Context

M-0308 ships the ritual. Until a seam calls it, it is a document nobody reaches:
`aiwfx-start-milestone`'s preflight still asks the reader to confirm each
criterion is concrete and testable with nothing behind the request, and
`wf-patch` step 1 asks the same of a gap in the same shape.

Both seams matter because the false claim can originate on either side. M-0307's
premise came from G-0541, a gap whose own title said the path resolved for two
of six kinds where measurement found none — so a patch closing a gap carries the
same exposure as a milestone implementing a spec.

## Acceptance criteria

### AC-1 — The milestone preflight names the ritual, not a bare judgment call

`aiwfx-start-milestone`'s preflight routes the reader to the ritual, and the
bullet that asked for the judgment without supplying a method is gone rather
than left standing beside it. Two surfaces stating the obligation is the
duplication D-0054 bans and the disagreement E-0081 spent an epic removing.

The existing structural test already scopes to this section through
`findStartMilestonePreflightSection`, and the heading-order assertion at
`TestAiwfxStartMilestone_M0105_AC4_WorkflowHeadingsInNewOrder` pins the workflow
shape — both move with this change rather than around it.

### AC-2 — A gap-closing patch measures the gap's claims before the change

`wf-patch` routes to the ritual when the patch closes a tracked gap, before the
change is written. Its step 1 asks the reader to state the goal in their own
words and to stop if they cannot; that judgment gets the same method the
milestone seam gets.

The gap's own claims are what the pass measures, and the record lands against
the gap id. A patch that closes no tracked item has no entity to measure and is
outside this criterion — the ritual is not made mandatory for a typo.

### AC-3 — No surface promises testable criteria without naming the method

No shipped surface claims to produce testable acceptance criteria while
supplying nothing that produces them. `aiwfx-plan-milestones` asserts it twice —
that each milestone has "clear, testable acceptance criteria", and that each AC
body carries "the testable contract" — and its own anti-pattern says why it
cannot deliver either: criteria written weeks before the work are usually wrong.
It hands the property forward to the seam that measures instead of asserting it.

The assertion enumerates shipped surfaces rather than checking the two known
ones, so a third surface making the same promise later fails the test.

## Constraints

- **The kernel does not change.** No check rule, finding code, config field or
  schema entry ships with this milestone.
- **Every `SKILL.md` edit lands with its referencing structural test**, per the
  repo's backstop policy. `aiwfx_start_milestone_test.go`,
  `aiwfx_plan_milestones_test.go`, `wf_patch_changelog_test.go` and
  `wf_patch_reconcile_test.go` already exist; extend them rather than minting
  parallel files.
- **Authoring lands under `internal/skills/embedded-rituals/`.**
  `.claude/skills/` is materialized and can be stale in any given clone
  (G-0504); read the embedded tree when comparing.
- **The ritual stays invocable on its own.** Wiring adds callers; it does not
  fold the ritual's content into a caller.

## Design notes

- The obligation lives in one place. A seam names the ritual and does not restate
  its steps, for the same reason `entity.RequiredSections` has one definition.
- `wf-patch`'s invocation is gap-scoped by design. The evidence for the method is
  entities carrying factual claims; a patch with no tracked item has none.

## Surfaces touched

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/`
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/`
- `internal/policies/` — the existing structural tests for each

## Out of scope

- The ritual's own content. M-0308 owns it.
- `aiwfx-start-epic`. E-0085 leaves the epic seam open deliberately — no evidence
  at that scale yet.
- A check rule that enforces the pass ran.
- G-0504's doctor-side drift detection, and the template-instruction half now
  folded into it.

## Dependencies

- M-0308 — the ritual must exist before a seam can name it.

## References

- E-0085 — parent epic
- G-0583 — the gap the epic closes
- G-0541, M-0307 — the gap-originated false claim
- E-0081, D-0054 — one owner, not a second copy

## Work log

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
