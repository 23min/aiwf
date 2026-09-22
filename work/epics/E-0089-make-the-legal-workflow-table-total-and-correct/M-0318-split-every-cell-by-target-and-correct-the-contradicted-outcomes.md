---
id: M-0318
title: Split every cell by target and correct the contradicted outcomes
status: in_progress
parent: E-0089
tdd: required
acs:
    - id: AC-1
      title: Rule carries a target and the enforced uniqueness key includes it
      status: met
      tdd_phase: done
    - id: AC-2
      title: Every legal cell's target agrees with entity.transitions, mechanically
      status: met
      tdd_phase: done
    - id: AC-3
      title: No declared outcome contradicts what the verb returns at that coordinate
      status: open
      tdd_phase: done
---
## Goal

Put the target in the cell key so the table can say where a verb takes an entity,
and correct the cells whose declared outcome the kernel contradicts.

## Closes

- G-0631 — distinguish same-state NoOp outcomes from illegal target transitions.
- G-0160 — enforce agreement between declared targets and the FSM's allowed edges.

## Context

The target distinguishes outcomes at the same origin and verb: promoting an epic
from `done` to `cancelled` is illegal, while promoting it to `done` is a NoOp
(G-0631). D-0077 puts that distinction in the cell key.

G-0160 names the missing per-edge drift check. Comparing the table's explicit
targets with the FSM makes a new edge require a declaration in the table; a driver
that derives its targets from the FSM cannot enforce that independent agreement.

Measured 2026-08-24: 61 cells over 99 coordinates, all 15 terminal-state `promote`
coordinates declared illegal. The expansion is bounded because the target is
derivable from `entity.transitions` wherever the cell is legal; the illegal and
NoOp rows are the ones needing judgement.

## Acceptance criteria

### AC-1 — Rule carries a target and the enforced uniqueness key includes it

`Rule` has a target field, and the key-uniqueness policy keys on it alongside
`Kind`, `FromState`, `Verb` and `Outcome`. Removing the target from the key makes
the policy fail. `spec.go`'s package documentation states the same key the policy
enforces.

### AC-2 — Every legal cell's target agrees with entity.transitions, mechanically

A policy compares each legal cell's target against `entity.AllowedTransitions` for
its kind and origin, in both directions: a cell naming a target the FSM does not
allow fails, and an FSM edge with no cell fails. Adding an edge to
`entity.transitions` without a cell reddens the suite.

### AC-3 — No declared outcome contradicts what the verb returns at that coordinate

For every cell, the outcome the table declares matches what the verb does. A cell
declaring illegal at a coordinate where the verb exits 0 fails. The 15
terminal-state `promote` coordinates carry a NoOp cell toward their own state and
an illegal cell toward every other target, and the count of corrected cells is
recorded with the command that produced it.

## Constraints

- **Derived rows are generated; judged rows are read.** The mechanical pass emits
  only what `entity.transitions` supports. Every other row is listed for hand-ruling
  and its count recorded — a row filled to complete the grid, with no argument for
  its outcome, is worse than the hole.
- **The kernel is the tiebreak for behavior.** Where table and kernel disagree about
  what a verb does, the table is corrected.
- **No coverage regression.** Per-cell coverage is measured before the split so the
  driver milestone has a baseline to compare against.

## Design notes

Target selection belongs to the table. The positive driver consumes each declared
target so splitting a cell cannot multiply its test cases by re-expanding the FSM.
M-0320 retains the negative-driver migration and rejection-layer reconciliation.

## Coverage measurement

Measured 2026-09-22 in the E-0089 worktree on Linux amd64 with Go 1.25.11. Run the
same command before the legal-row split and against the AC-2 implementation:

```bash
go test -json -count=1 -parallel 8 \
  -run 'TestM0124_PositiveDriver_LegalCells|TestM0125_AC2_NegativeDriver_VerbTimeRejection|TestM0125_AC3_NegativeDriver_CheckTimeRejection' \
  ./internal/policies
```

Expected: every driver passes, with no previously covered subtest removed.
Observed: both runs exited 0. Counted JSON events with `Action == "pass"` and a
`Test` containing `/`, grouped by the parent test name:

| Driver | Before | After |
|---|---:|---:|
| Positive legal cells | 40 | 40 |
| Verb-time rejection | 28 | 28 |
| Check-time rejection | 2 | 2 |

The sets of passing subtest names were identical: no additions or removals.
The one-off FSM-derived expansion emitted 40 explicit legal rows from 31 coarse
rows. These are coverage measurements, not the corrected-outcome count AC-3 owns.

The bidirectional check was also run with a Go overlay adding `proposed` to the
epic `active` state's allowed targets in `internal/entity/transition.go`:

```bash
go test -overlay=/tmp/aiwf-m0318-ac2-vacuity-yrekd4zv/added-fsm-edge.json \
  -count=1 -parallel 8 \
  -run '^TestM0318_AC2_LegalTargetsAgreeWithFSM$' ./internal/policies
```

Expected: a missing-cell failure. Observed: exit 1 with
`FSM edge (epic, "active", "proposed") has no legal rule`.
The overlay leaves the checked-out FSM unchanged.

## Out of scope

- Applicability and totality enforcement — its own milestone.
- The `authorize` cells — moved out of the table by its own milestone, so this one
  leaves them where they are.
- The `tdd_phase` same-phase question (G-0458).

## Dependencies

None. First milestone of the epic; everything else is expressed in terms of the key
it establishes.

## References

- D-0077 — the ruling this implements
- G-0631 — the contradicted outcomes AC-3 corrects
- G-0160 — the per-edge drift AC-2 closes
- M-0281 — the same-state NoOp convention the table must now express
- `internal/workflows/spec/spec.go`, `rules.go`; `internal/policies/m0123_ac2_rules_test.go`
