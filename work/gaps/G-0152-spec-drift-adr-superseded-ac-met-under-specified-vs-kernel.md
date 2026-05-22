---
id: G-0152
title: 'spec drift: ADR.superseded + AC.met under-specified vs kernel'
status: addressed
prior_ids:
    - G-0151
addressed_by:
    - M-0124
---
## Problem

Two Legal cells in M-0123's `spec.Rules()` under-specify their preconditions relative to what the kernel actually enforces. The driver test built in M-0124/AC-3 surfaces this drift: the spec says "Legal", the kernel demands additional flags or fixture state that no precondition cell mentions, and the driver either fails or papers over with operator-path setup.

### Drift 1 — `(adr, accepted, promote) → superseded`

Spec cell at `internal/workflows/spec/rules.go` carries no preconditions. The kernel's `adr-supersession-mutual` finding enforces `--superseded-by <ADR-id>` at verb-time:

```
aiwf promote ADR-0001 superseded
# error: promoting an ADR to "superseded" requires --superseded-by <ADR-id>
#        so the adr-supersession-mutual rule is satisfied
```

Comparable atom already exists on the gap side: `(gap, open, promote, self.addressed_by non-empty) → Legal` paired with `(gap, open, promote, self.addressed_by == "") → Illegal gap-resolved-has-resolver`. ADR supersession needs the same pair.

### Drift 2 — `(ac, open, promote, self.evidence non-empty) → met`

Spec cell carries `self.evidence non-empty` but says nothing about `tdd_phase`. The kernel's `acs-tdd-audit` fires when `parent.tdd == required AND self.tdd_phase != done`:

```
aiwf promote M-0001/AC-1 met
# acs-tdd-audit (error): M-0001/AC-1 status: met under tdd: required
#                        but tdd_phase is red (expected done)
```

The Illegal companion exists at `(ac, open, promote, parent.tdd==required AND self.tdd_phase!=done) → acs-tdd-audit`. The Legal cell needs the converse — but since the current Predicate model is single-atom (implicit-AND across a `[]Predicate` list), encoding the disjunction `parent.tdd != required OR self.tdd_phase == done` requires splitting the Legal cell into two:

1. `(ac, open, promote, self.evidence non-empty, parent.tdd != required) → Legal`
2. `(ac, open, promote, self.evidence non-empty, parent.tdd == required, self.tdd_phase == done) → Legal`

(YAGNI: split into two cells rather than widen Predicate to support disjunction.)

## Why this matters

- M-0124/AC-3's per-cell positive driver is the first test that crosses spec → kernel via real subprocess. Spec under-specification surfaces as either test failure or driver workarounds. Workarounds skimp.
- M-0125 (negative cell coverage) will hit the same drift from the Illegal side — it needs to test the Illegal companions fire when preconditions are violated. The companions are present for `acs-tdd-audit` but absent for `adr-supersession-mutual`.
- The Predicate Subject vocabulary (`self.target-state`, `self.evidence`, `self.addressed_by`, `self.tdd_phase`, `parent.tdd`, `any-child.status`, `any-child-ac.status`, `all-children-acs.status`) lacks `self.superseded_by`. Widening adds one atom + one ~5-line case in `spec.EvaluatePredicate`.

## Fix outline

1. Add `self.superseded_by` to `Predicate.Subject` vocabulary (closed-set documented in `internal/workflows/spec/spec.go`).
2. Handle the new Subject in `spec.EvaluatePredicate` (`internal/workflows/spec/evaluate.go`): same shape as `self.addressed_by` — read the entity's `SupersededBy` field, apply `non-empty` / `==` ops via existing helpers.
3. Update `adrRules()` in `internal/workflows/spec/rules.go`:
   - Add precondition `self.superseded_by non-empty` to the existing `(adr, accepted, promote)` Legal cell.
   - Add new Illegal cell `(adr, accepted, promote, self.superseded_by == "") → adr-supersession-mutual` (VerbTime, BlockingStrict).
4. Update `acRules()`: split the `(ac, open, promote, evidence non-empty)` Legal cell into the two converse-of-acs-tdd-audit shapes per "Drift 2" above.
5. Update `spec.EvaluatePredicate` tests in `internal/workflows/spec/evaluate_test.go` to cover the new atom.
6. Verify M-0123's drift policies under `internal/policies/` still pass (they should — the changes are additive to the cells, not restructuring).
7. Remove `prepareKernelPreconditions` from `internal/policies/m0124_positive_driver_test.go`; the now-encoded preconditions flow through the existing predicate-materialization loop (`self.superseded_by` becomes verb-arg-shaped like `self.addressed_by`).

## Where to fix

Within M-0124 itself — the changes are bounded (~30 LOC across `rules.go` + `evaluate.go` + targeted tests) and removing the M-0124 driver's operator-path setup IS the gain. Treating this as "fix M-0123 incompleteness mid-M-0124" preserves the "design first, fix kernel/spec if it disagrees" discipline; deferring to a separate fix milestone bakes the drift into the driver semi-permanently.

## Discovered in

M-0124 / AC-3 (Per-cell positive driver). The driver iterates `spec.Rules()` Legal cells; two cells failed against the real binary until operator-path setup was added, and that setup is the skimp this gap removes.

## Status

`open` — fixing inline within M-0124/AC-3's red→green cycle.
