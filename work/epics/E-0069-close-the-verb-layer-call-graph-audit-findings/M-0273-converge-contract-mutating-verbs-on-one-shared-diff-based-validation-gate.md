---
id: M-0273
title: Converge contract-mutating verbs on one shared diff-based validation gate
status: in_progress
parent: E-0069
tdd: required
acs:
    - id: AC-1
      title: a shared gate reports only findings introduced by the projected mutation
      status: open
      tdd_phase: green
    - id: AC-2
      title: bind, unbind, recipe install, and recipe remove route through the shared gate
      status: open
      tdd_phase: red
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

### AC-1 — a shared gate reports only findings introduced by the projected mutation

A new `internal/verb/contractgate.go` holds `contractMutationGate(t
*tree.Tree, current, next *aiwfyaml.Contracts, repoRoot string)
[]check.Finding`: it runs `contractcheck.Run` once against `current`
and once against `next`, then returns the findings present in the
`next` run that are not already present in the `current` run — a true
multiset before/after diff, not an id-filtered subset. A finding
present in both runs (a pre-existing issue on an entry the mutation
didn't touch) is excluded; a finding introduced by the mutation, on
any entry, is returned; a finding the mutation *resolves* (present
before, absent after) is not reported — the gate only reports
additions.

Evidence: `internal/verb/contractgate_test.go` exercises the diff
directly — a mutation that changes nothing produces zero introduced
findings even when `current` already carries pre-existing findings; a
mutation that adds an entry with a missing schema/fixtures path
surfaces exactly those two findings; a pre-existing finding on an
untouched entry is excluded from the introduced set even when the
mutation is otherwise "dirty" elsewhere.

### AC-2 — bind, unbind, recipe install, and recipe remove route through the shared gate

`ContractBind`, `ContractUnbind`, `RecipeInstall`, and `RecipeRemove`
(`internal/verb/contractbind.go`, `contractrecipe.go`) all call
`contractMutationGate` before writing, in place of bind's existing
id-filtered `contractCheckForBinding` (removed) and unbind/recipe-
install/recipe-remove's previous lack of any contract-check gate.
Each verb function gains the `t *tree.Tree` and `repoRoot string`
parameters needed to run the check; the CLI dispatchers that didn't
already load a tree (`internal/cli/contract/unbind.go`, `recipes.go`'s
install and remove paths) now do, mirroring `bind.go`'s existing
`tree.Load` call. Recipe remove keeps its `bindingsReferencing`
referential-integrity error ahead of the gate call, per the
milestone's constraint — the gate is an additional safety net there,
not a replacement for that specific error message.

Evidence: the migrated `internal/verb/contractbind_test.go` /
`contractrecipe_test.go` suites (updated call sites, still green) plus
new coverage proving the previously-ungated verbs' safety net is live
— a case constructed so unbind's or recipe-remove's projected mutation
would introduce a contract-config finding, confirming the gate
actually fires rather than just being wired in dead. `go vet`/`go
build` clean repo-wide; the CLI-level `internal/cli/integration`
contract-verb tests (`single_commit_invariant_test.go`,
`trailer_shape_test.go`, `remaining_verbs_diag_test.go`,
`verb_metadata_test.go`) pass unmodified, confirming existing verb
envelopes and exit codes are unchanged for normal operation.

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
