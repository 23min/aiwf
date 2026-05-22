---
id: G-0154
title: cellcoverage.lookupComposite duplicated in driver test (export + reuse)
status: open
discovered_in: M-0124
---
## Problem

The composite-id lookup helper exists in two places with byte-identical bodies:

- `internal/cellcoverage/fixture.go::lookupComposite` (package-private, line ~582)
- `internal/policies/m0124_positive_driver_test.go::lookupCompositeForDriver` (test-file-local, line ~487)

Both functions resolve `M-NNNN/AC-N` to `(parent, AC slot)` via `tree.ByID` and a slot iteration. The driver-side helper's comment claims it exists "so the driver doesn't depend on cellcoverage's internals" — but the driver already imports `cellcoverage.CellFixture`, `BringOpts`, `SatisfyPredicate`, etc. The justification is thin.

Surfaced by the M-0124 reviewer agent's audit (post-wrap, pre-merge).

## Why it matters

Small but real risk: 15 LOC of duplicated logic that would drift independently if one branch changed and the other didn't. Future readers grepping for `lookupComposite` find two implementations and have to compare them line by line.

## Fix outline

Either:

1. **Export** `cellcoverage.LookupComposite` (capitalize, document in package doc) and delete `lookupCompositeForDriver` from `m0124_positive_driver_test.go`. Driver consumes the package symbol like any other helper.
2. **Accept** the duplication and edit the driver-test comment to drop the misleading justification. Smaller diff but the duplication stays.

Recommend (1) — the duplication is small enough that exporting is cheap, and the symmetric usage pattern is more discoverable.

## Discovered in

M-0124 (reviewer-agent audit pre-merge).

## Status

`open`.
