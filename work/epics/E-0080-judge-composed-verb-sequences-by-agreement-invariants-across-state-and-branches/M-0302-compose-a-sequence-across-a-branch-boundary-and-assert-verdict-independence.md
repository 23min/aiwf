---
id: M-0302
title: Compose a sequence across a branch boundary and assert verdict independence
status: draft
parent: E-0080
depends_on:
    - M-0300
tdd: required
acs:
    - id: AC-1
      title: A deterministic scenario runs a sequence's steps on both sides of a branch
      status: cancelled
    - id: AC-2
      title: A sequence's verdict does not depend on which branch ran which step
      status: cancelled
    - id: AC-3
      title: The scenario runs untagged and its per-push cost is recorded
      status: open
---
## Goal

Let a composed sequence cross a branch boundary, and assert the property that
becomes reachable once it can: a sequence's verdict does not depend on which
branch ran which step.

## Context

No scenario in the catalog composes a sequence across a branch boundary. Every
two-branch scenario there is about contention — concurrent allocation, worktree
races, reallocate collisions — so no reference is ever authored in one context
and judged in another, which is the composition the whole cross-branch tier
exists for.

M-0300 deliberately left the branch-independence property out. Asserting it
before a sequence could generate the condition would have been a test that
passes because nothing reaches it, which is the vacuity the epic's success
criteria rule out.

## Approach

Add a scenario that cuts a branch, runs some steps of a sequence on each side,
merges, and evaluates the oracle throughout. The scenario is deterministic — no
concurrency, no timing, no observation window — so its verdict is a fact about
aiwf rather than about the machine.

The branch-independence property compares the verdict of a sequence against the
verdict of the same steps distributed differently across the two branches. As
with the M-0300 properties, it compares two observations and holds no model of
which verdict is correct.

## Acceptance criteria

Each criterion below is asserted by the harness itself, not by review.

### AC-1 — A deterministic scenario runs a sequence's steps on both sides of a branch

A registered scenario cuts a branch, runs some steps of a composed sequence on
each side, merges, and evaluates the oracle after every step on both sides and
after the merge.

Determinism is part of the contract, not an accident of the implementation: no
concurrently scheduled subprocesses, no wall-clock dependence, no observation
window. A slower machine delays the verdict; it never changes it.

This is the first scenario in the catalog where a reference is authored in one
context and judged in another. Every existing two-branch scenario is about
contention — concurrent allocation, worktree races, reallocate collisions — and
none composes a sequence across the boundary.

### AC-2 — A sequence's verdict does not depend on which branch ran which step

The harness evaluates the same set of steps under more than one distribution
across the two branches and requires the verdicts to agree. A verdict that
changes when a step moves from one branch to the other is reporting on the
choreography rather than on the tree.

The property compares two observations and holds no model of which verdict is
correct, so it is independent of the reference-resolution tier policy. It holds
under ADR-0030 as written and under the published-versus-local split ADR-0041
introduces — which is exactly why it can be built while G-0556 is still in
flight.

### AC-3 — The scenario runs untagged and its per-push cost is recorded

The scenario carries no `//go:build stress` tag, runs inside the ordinary
`go test ./...`, and its added wall time on the every-push path is measured and
recorded in this milestone's Validation section.

Per `CLAUDE.md`, lane choice is a cost decision rather than a correctness one:
a scenario goes tagged when it drives real concurrently scheduled subprocesses
and would slow every push. This one does not, so untagged is the correct
placement and D-0063 accepts its cost deliberately.

Recording the number is what keeps that acceptance honest. A cost nobody
measured is a cost that gets rediscovered later as pressure to tag the scenario,
which would hide the class it exists to catch.

## Constraints

- **Untagged, in the default lane.** The scenario drives no concurrently
  scheduled subprocesses, so by the lane rules in `CLAUDE.md` it stays untagged
  and runs on every push. D-0063 accepts that per-push cost deliberately rather
  than discovering it later.
- **Deterministic.** No wall-clock dependence and no observation window: a
  slower machine delays the verdict, it never changes it (G-0468).
- **The property compares observations**, never an expected verdict the harness
  computed (D-0063).

## Design notes

- D-0063's Consequences state the lane placement for this scenario explicitly,
  and give the reason: deterministic scenarios land untagged.
- The reference-resolution tiers this exercises are ADR-0030's cross-branch view
  and ADR-0041's published-versus-local split. The property is independent of
  both — it holds under either policy, which is what makes it safe to build
  while G-0556 is still in flight.

## Surfaces touched

- `cmd/stresstest` — scenario registry and the walker's branch handling
- `internal/stresstest` — the new scenario driver

## Out of scope

- Changing the tier policy of ADR-0030 or ADR-0041.
- Multi-host or multi-clone choreography. G-0564 carries that.
- Widening the mutation space — that is its own milestone and this one composes
  whatever axes exist when it lands.

## Dependencies

- M-0300 — the oracle seam this registers a third property into.
- D-0063, accepted.

## References

- D-0063 — the accepted direction and this scenario's lane placement.
- ADR-0030, ADR-0041 — the tiers the scenario exercises and the property is
  independent of.
- G-0556 — the in-flight classification change this must not encode.
- G-0468 — the oracle-shape precedent.
- `CLAUDE.md` — the tagged-versus-untagged lane rules.

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
