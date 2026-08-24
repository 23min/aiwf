---
id: M-0319
title: Move authorize to GlobalRules and sweep the stale entries there
status: draft
parent: E-0089
tdd: required
acs:
    - id: AC-1
      title: authorize appears in no cell and its kind restriction still fails when removed
      status: open
    - id: AC-2
      title: The retired finding-code entries and their dead constructors are gone
      status: open
---
## Goal

Take `authorize` out of the cell table and express its kind restriction where its
other rules already live, sweeping the retired entries sitting there while the file
is open.

## Context

`authorize` binds a scope to a branch. It has no `FromState` semantics: its four
declared cells all say "wrong kind" and none says anything about state, while the
legal side — epic and milestone, the verb's actual purpose — is absent from all
eight of its coordinates. Of the 50 undeclared coordinates measured on 2026-08-24,
29 are `authorize`.

D-0077 rules that it leaves the cell table. The destination is not new:
`GlobalRules()` already holds four `authorize` rules —
`provenance-authorization-out-of-scope`, `branch-context-required`,
`branch-not-found` and `rung-pair-illegal` — as cross-cutting preconditions with no
cell coordinate, which is the shape ADR-0013 created. The four cells in `Rules()`
are the outliers.

Two of those global entries are already stale. G-0417 records that
`branch-not-found` was subsumed by `rung-pair-illegal` per D-0018, leaving a dead
code path in `internal/verb/authorize.go` and entries citing the retired code in
`GlobalRules()`, in `branch/rules.go`, and in a policy test's keyword map. Adding
entries beside them without sweeping would leave the destination in worse shape
than the origin.

## Acceptance criteria

### AC-1 — authorize appears in no cell and its kind restriction still fails when removed

`spec.Rules()` contains no cell whose verb is `authorize`. The restriction that
`authorize` refuses non-epic and non-milestone entities is expressed in
`GlobalRules()` and is covered by a test that fails when the rule is removed.
D-0007 remains the record the rule cites.

### AC-2 — The retired finding-code entries and their dead constructors are gone

No surface names `branch-not-found`: not `GlobalRules()`, not
`internal/workflows/spec/branch/rules.go`, not the policy keyword map. The
unreferenced constructors in `internal/verb/authorize.go` are removed. A search for
the retired code across `internal/` returns nothing, and the command is recorded.

## Constraints

- **Move, do not weaken.** The kind restriction must still refuse the same four
  kinds after the move; the test that proves it fails when the rule is deleted.
- **Sweep only what G-0417 names.** Other entries in `GlobalRules()` are not this
  milestone's business.
- **No behavior change.** The verb refuses exactly what it refused before; only
  where the refusal is recorded changes.

## Design notes

`GlobalRules()` holds four entries today, so this roughly doubles it. That is the
intended direction under ADR-0013 — a precondition carrying no cell coordinate
belongs there — and it is worth stating plainly, since a reader meeting a grown
global list should not read it as the cell table leaking.

## Out of scope

- Anything about `authorize`'s branch-rung machinery beyond the retired code
  G-0417 names.
- The cell key change — its own milestone, and this one is independent of it.

## Dependencies

None. Independent of the key change and runnable in either order.

## References

- D-0077 — the ruling this implements
- D-0007 — the kind restriction being relocated
- ADR-0013 — global rules as the home for coordinate-free preconditions
- G-0417 — the stale entries and dead code swept here
- D-0018 — the supersession that retired `branch-not-found`
