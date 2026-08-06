---
id: M-0301
title: Widen the walker's mutation space beyond status
status: draft
parent: E-0080
depends_on:
    - M-0300
tdd: required
acs:
    - id: AC-1
      title: A sequence seeds acceptance criteria and mutates them through real verbs
      status: open
    - id: AC-2
      title: A sequence sets areas and priorities and rewrites bodies
      status: open
    - id: AC-3
      title: An axis the kernel cannot express is recorded against its owning gap
      status: open
---
## Goal

Widen what a composed sequence may mutate beyond `status`, so the invariant
oracle judges reference state, classification state, and body content rather
than the six FSMs alone.

## Context

M-0300 gave the walker an oracle that costs nothing per new axis of mutation.
This milestone spends that: the walker moves `status` and nothing else today, so
the reachable space is statuses, not references, areas, priorities, edges or
bodies — and a defect in any of those is unreachable by construction.

D-0063 ordered these two deliberately. A wider state space under the old
monotonic oracle would have bought reachability without judgment, which is why
the oracle landed first and the widening lands here.

## Approach

Extend the walker's weighted operation table with the verbs that mutate the
other axes, seeding whatever state each needs first: acceptance criteria via
`aiwf add ac` and the phase promote, classification via `aiwf set-area` and
`aiwf set-priority`, body content via `aiwf edit-body`. Each new operation
reports its own legality verdict as the existing ones do, and the oracle from
M-0300 evaluates unchanged after every step.

Two axes are expected to refuse. Declaring an edge meets the milestone-only
restriction on `depends_on`, and the set-at-create reference fields have no
mutation verb at all. Where an axis cannot widen, the constraint is recorded
against the gap that owns it rather than routed around with a hand-written file
edit — a walker that hand-edits state no verb can produce would be testing a
tree the kernel cannot reach.

## Acceptance criteria

Each criterion below is asserted by the harness itself, not by review.

### AC-1 — A sequence seeds acceptance criteria and mutates them through real verbs

A composed sequence seeds acceptance criteria on a milestone it created and then
mutates them: `aiwf add ac` to seed, the phase promote to move one through its
TDD phases, and `aiwf milestone tdd` to flip the policy underneath. Each
operation reports its own legality verdict exactly as the existing `status`
operations do.

The walker seeds none today, which is why the acceptance-criterion composition
invariant is structurally unreachable — D-0063 records this. Reaching that state
is the point of this criterion; asserting the invariant over it belongs to a
later milestone.

### AC-2 — A sequence sets areas and priorities and rewrites bodies

A composed sequence sets and changes an entity's area and priority, and rewrites
a body, through `aiwf set-area`, `aiwf set-priority` and `aiwf edit-body`. The
sequence's repository declares the area members those operations validate
against, so the walker exercises both the accepted and the refused path.

Body edits matter beyond coverage: a body is where prose references live, so
mutating one is what lets a reference be authored in one state and judged in
another — the composition the read-path agreement property from M-0300 is
waiting for.

### AC-3 — An axis the kernel cannot express is recorded against its owning gap

An axis the kernel cannot express is recorded against the gap that owns it, and
the walker does not reach the state by another route.

Two are expected. Declaring a cross-kind edge meets `depends_on`'s
milestone-only restriction (G-0073); the set-at-create reference fields have no
mutation verb (G-0168). For each, the walker carries a named, reasoned refusal
pointing at the owning gap rather than a hand-written file edit.

The distinction is load-bearing. A walker that hand-edits state no verb can
produce tests a tree the kernel cannot reach, so a property passing over it says
nothing about real repositories — and a property failing over it sends the
implementer after a defect that cannot occur.

## Constraints

- **Every mutation runs through a real verb.** No hand-written frontmatter or
  file edits to reach a state, even when a verb is missing: an unreachable state
  is a finding about the kernel, not an obstacle to route around.
- **The oracle is not modified here.** If a widened axis makes a property fail,
  the failure is a result, not a reason to weaken the property.
- **No property measures the runner** (G-0468), and no new operation asserts an
  expected end state the harness computed (D-0063).

## Design notes

- D-0063's Consequences name both ceilings this will meet: a widened walker that
  declares edges hits `depends_on`'s milestone-only restriction (G-0073), and
  there is no verb for the set-at-create reference fields (G-0168).
- Seeding acceptance criteria is what unblocks M-0300's successor invariant on
  AC composition; D-0063 records that the walker seeds none today, which is why
  that invariant is structurally unreachable.

## Surfaces touched

- `cmd/stresstest` — the walker's operation table
- `internal/stresstest` — scenario drivers

## Out of scope

- Adding the missing verbs. G-0168 owns the set-at-create fields; widening
  `depends_on` beyond milestone-to-milestone is G-0073.
- Asserting the acceptance-criterion composition invariant. This milestone makes
  it reachable; the assertion is its own milestone.
- Crossing a branch boundary.

## Dependencies

- M-0300 — the invariant oracle. Widening ahead of it is the combination D-0063
  rejects.
- D-0063, accepted.

## References

- D-0063 — the accepted direction, including both ceilings named here.
- G-0073, G-0168 — the kernel gaps that bound how far this widens.
- G-0468 — the oracle-shape precedent.
- E-0062, E-0071 — the walker and the `milestone tdd` flips it already drives.

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
