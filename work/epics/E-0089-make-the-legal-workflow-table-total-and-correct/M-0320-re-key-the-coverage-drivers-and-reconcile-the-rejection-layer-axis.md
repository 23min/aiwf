---
id: M-0320
title: Re-key the coverage drivers and reconcile the rejection-layer axis
status: draft
parent: E-0089
depends_on:
    - M-0318
tdd: required
acs:
    - id: AC-1
      title: Both coverage drivers key on the target, with coverage measured before and after
      status: open
    - id: AC-2
      title: Each cell's rejection layer matches where the kernel actually refuses
      status: open
---
## Goal

Re-key the positive and negative cell drivers onto the target, and settle whether
each cell's rejection layer names where the kernel actually refuses.

## Context

M-0124 and M-0125 drive per-cell coverage from `(Kind, FromState, Verb)`. Once the
target joins the key they must follow, and the follow is not mechanical: M-0124
derives targets from `entity.AllowedTransitions` today because the cell did not
carry them, and that derivation becomes redundant — or wrong — when the cell names
its own target.

G-0166 names a second divergence the re-key is the moment to settle. Two cells
declare `RejectionLayerCheckTime` — the gap `open → addressed` cell missing a
resolver, and the AC `open → met` cell under `tdd: required` — while the kernel
refuses both at verb time, one via a hand-rolled guard in `promote.go`, one via a
pre-write projection in `ac.go`. The gap argues the kernel being stricter is
design-aligned; what is not aligned is the table claiming an axis the kernel does
not use, because M-0125's check-time driver expects the verb to succeed first.

## Acceptance criteria

### AC-1 — Both coverage drivers key on the target, with coverage measured before and after

The positive and negative drivers enumerate cells by their declared target rather
than deriving targets separately. Per-cell coverage is measured before the re-key
and after, both with the command that produced the numbers, and no cell covered
before is uncovered after.

### AC-2 — Each cell's rejection layer matches where the kernel actually refuses

For every illegal cell, the declared rejection layer agrees with where the kernel
refuses — a cell declaring check-time rejection at a coordinate the verb refuses
before writing fails. The two cells G-0166 names are corrected, or the gap is
re-scoped with the argument for why the axis should stay as declared.

## Constraints

- **No silent coverage loss.** A cell that stops being exercised is a failure of
  this milestone even if the suite is green; the before-and-after measurement is the
  evidence.
- **Do not weaken the kernel to match the table.** Where the kernel refuses earlier
  than the table declares, the table is corrected, not the guard.

## Design notes

The check-time-versus-verb-time question has a third possible answer beyond the two
in G-0166: the cell may legitimately carry both, if the verb refuses and the check
rule still fires as a backstop for entities that arrive by another path. If that is
the answer, the axis needs a value for it rather than a choice between the two.

## Out of scope

- Adding coverage for coordinates that had none — that is the totality milestone.
- The `tdd_phase` same-phase question (G-0458).

## Dependencies

- M-0318 — the key the drivers are re-keyed onto.

## References

- D-0077 — the ruling behind the key change
- G-0166 — the rejection-layer divergence AC-2 settles
- `internal/policies/m0124_positive_driver_test.go`, `m0125_negative_driver_test.go`
