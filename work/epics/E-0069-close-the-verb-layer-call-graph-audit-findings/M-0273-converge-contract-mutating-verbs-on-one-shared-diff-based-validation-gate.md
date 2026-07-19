---
id: M-0273
title: Converge contract-mutating verbs on one shared diff-based validation gate
status: draft
parent: E-0069
tdd: required
---
## Goal

Give the contract-mutating verbs one shared validation gate: findings
introduced by a mutation are computed as a before/after diff of contract-check
findings on the projected config, and all four verbs route through it.

## Context

The audit found three unrelated gate styles across bind, recipe install, and
recipe remove (finding F10 of `docs/initiatives/verb-layer-cleanup.md`), with
unbind ungated. The convergence decision (see References) chose a diff-based
gate because id-filtered scoping cannot generalize to verbs that mutate the
validators map. Bind's current filter is not a true before/after diff; this
milestone makes the diff the shared semantics.

## Acceptance criteria

## Constraints

- Test-first per AC (`tdd: required`).
- Remove keeps its precise "referenced by bindings: <ids>" error on top of the
  shared gate — the gate replaces gate *logic*, not better error messages.
- Existing verb envelopes and exit codes unchanged; pre-existing findings on
  untouched entries never block a mutation (the diff guarantees this by
  construction).

## Design notes

- Gate shape: run the contract check on current and projected configs, report
  only findings present in the projection and absent from current.
- The converged-gate decision entity in References carries the full rationale.

## Out of scope

- `contract verify`'s external-validator pipeline (deliberately separate).
- Any change to what the underlying contract check validates.

## Dependencies

- None — parallel-safe with the sibling E-0069 milestones.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F10; D-0041, the convergence decision
  entity; `internal/verb/contractbind.go`, `internal/verb/contractrecipe.go`.

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
