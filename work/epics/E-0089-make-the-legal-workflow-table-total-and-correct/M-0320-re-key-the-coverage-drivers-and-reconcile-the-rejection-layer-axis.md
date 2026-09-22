---
id: M-0320
title: Re-key the coverage drivers and reconcile the rejection-layer axis
status: in_progress
parent: E-0089
depends_on:
    - M-0318
tdd: required
acs:
    - id: AC-1
      title: Both coverage drivers key on the target, with coverage measured before and after
      status: cancelled
    - id: AC-2
      title: Each cell's rejection layer matches where the kernel actually refuses
      status: open
      tdd_phase: red
---
## Goal

Complete the coverage drivers' use of declared targets, and settle whether each
cell's rejection layer names where the kernel actually refuses.

## Context

M-0318 makes both drivers consume declared targets as a prerequisite to testing
the split rows. Its coverage measurements provide this milestone's baseline.
The driver migration is covered by M-0318's completed criteria. This milestone
reconciles the rejection-layer axis and compares exercised requests against the
recorded baseline; it must not lose coverage when a cell changes drivers.

G-0166 names a second divergence the re-key is the moment to settle. Two cells
declare `RejectionLayerCheckTime` — the gap `open → addressed` cell missing a
resolver, and the AC `open → met` cell under `tdd: required` — while the kernel
refuses both at verb time, one via a hand-rolled guard in `promote.go`, one via a
pre-write projection in `ac.go`. The gap argues the kernel being stricter is
design-aligned; what is not aligned is the table claiming an axis the kernel does
not use, because M-0125's check-time driver expects the verb to succeed first.

## Acceptance criteria

### AC-1 — Both coverage drivers key on the target, with coverage measured before and after

The driver migration and its test-first record belong to M-0318/AC-2 and
M-0318/AC-3. This criterion duplicates that completed implementation. The
before-and-after coverage comparison remains required under AC-2 and the
no-silent-coverage-loss constraint below.

### AC-2 — Each cell's rejection layer matches where the kernel actually refuses

For every illegal cell, the declared rejection layer agrees with where the kernel
refuses — a cell declaring check-time rejection at a coordinate the verb refuses
before writing fails. The two cells G-0166 names are corrected, or the gap is
re-scoped with the argument for why the axis should stay as declared.

Measure per-request driver coverage before and after the layer correction using
the same command, record the results and environment, and account for every
request that moves to a different driver. Preserve the forced gap-resolution
backstop check as well as unforced refusal coverage.

## Constraints

- **No silent coverage loss.** A cell that stops being exercised is a failure of
  this milestone even if the suite is green; the before-and-after measurement is the
  evidence.
- **Do not weaken the kernel to match the table.** Where the kernel refuses earlier
  than the table declares, the table is corrected, not the guard.

## Design notes

`RejectionLayer` describes where the ordinary request represented by a cell is
refused. A check rule that also catches invalid state introduced through another
path does not change that request's rejection layer. Preserve those backstop
checks independently; no combined layer value is needed.

## Out of scope

- Adding coverage for coordinates that had none — that is the totality milestone.
- TDD phase convergence (G-0458) — implemented and tested by M-0318.

## Dependencies

- M-0318 — the key the drivers are re-keyed onto.

## References

- D-0077 — the ruling behind the key change
- G-0166 — the rejection-layer divergence AC-2 settles
- `internal/policies/m0124_positive_driver_test.go`, `m0125_negative_driver_test.go`

## Deferrals

- G-0166 — modeling `milestone tdd` data-field mutations remains outside the
  status-transition table. Correcting its transition cells does not close this
  remainder; the gap must retain that scope at wrap.

## Coverage baseline

Measured on 2026-09-22 on the E-0089 epic branch after M-0319, in the Linux amd64
devcontainer with Go 1.25.11:

```bash
go test -json -count=1 -parallel 8 \
  -run 'TestM0124_PositiveDriver_LegalCells|TestM0125_AC2_NegativeDriver_VerbTimeRejection|TestM0125_AC3_NegativeDriver_CheckTimeRejection|TestM0319_AC1_ExcludedKindsRefuseWithoutSideEffects|TestM0318_AC3_CheckTimeCellsAlsoRefuseUnforcedVerb' \
  ./internal/policies
```

Expected: all selected tests pass and the transition requests measured in
M-0318 remain exercised. Observed: exit 0. Counting passing JSON subtest events
by parent test yields 40 positive cases, 60 verb-time negative cases, 2 cases in
the check-time driver, 4 dedicated wrong-kind authorization cases, and 2
supplemental unforced-refusal cases. The four authorization cases moved out of
the transition driver in M-0319; their dedicated binary tests retain coverage.

The check-time driver's passing cases do not both demonstrate post-write
rejection: the gap case uses force, and the AC case asserts verb-time refusal.
The layer correction must preserve these actual requests, not merely their count.
