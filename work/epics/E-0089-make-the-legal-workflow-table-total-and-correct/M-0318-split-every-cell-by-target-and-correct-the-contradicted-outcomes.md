---
id: M-0318
title: Split every cell by target and correct the contradicted outcomes
status: draft
parent: E-0089
tdd: required
acs:
    - id: AC-1
      title: Rule carries a target and the enforced uniqueness key includes it
      status: open
    - id: AC-2
      title: Every legal cell's target agrees with entity.transitions, mechanically
      status: open
    - id: AC-3
      title: No declared outcome contradicts what the verb returns at that coordinate
      status: open
---
## Goal

Put the target in the cell key so the table can say where a verb takes an entity,
and correct the cells whose declared outcome the kernel contradicts.

## Context

`Rule` carries `Kind`, `FromState` and `Verb` and no target. One cell therefore
covers every target reachable from that origin, which makes some coordinates
inexpressible: `epic|done|promote` is illegal toward `cancelled` and a NoOp toward
`done`, and the single declared outcome — illegal — is wrong for the second
(G-0631).

D-0077 rules that the target joins the key. G-0160 reached the same place from the
coverage side: without a target on the cell, a new FSM edge to an existing state
gets no coverage and no drift signal, and its fix outline names splitting cells per
target as the remedy.

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

The enforced key today is (`Kind`, `FromState`, `Verb`, `Outcome`) plus
preconditions — `m0123_ac2` permits complementary cells at one coordinate — while
`spec.go`'s package documentation claims the triple alone. AC-1 settles that
disagreement in passing rather than leaving two contradictory statements about the
same invariant.

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
