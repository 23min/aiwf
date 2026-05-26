---
id: M-0145
title: Implement scope-reach in EvaluatePredicate with verb-invocation context
status: in_progress
parent: E-0037
depends_on:
    - M-0144
tdd: required
acs:
    - id: AC-1
      title: EvaluatePredicate evaluates scope-reach for reachable and unreachable inputs
      status: met
      tdd_phase: done
    - id: AC-2
      title: scope-reach predicate agrees with tree.ReachesScope across the fixture set
      status: met
      tdd_phase: done
    - id: AC-3
      title: EvalContext carries the scope-entity and target the scope-reach arm needs
      status: met
      tdd_phase: done
---
## Goal

Implement the `scope-reach` subject in `EvaluatePredicate`, threading the actor's active-scope-entity + target through `EvalContext`, **delegating to `tree.ReachesScope`** (the M-0141 source of truth) rather than re-deriving D-0006.

## Context

`scope-reach` is in the spec's `Predicate` subject vocabulary (`spec.go`) but `EvaluatePredicate` (`evaluate.go`) returns `unknown subject` for it. The other four subjects are entity-side (`self.`/`parent.`/`all-children.`/`any-child.`); `scope-reach` is the first that needs *verb-invocation* context — the actor's active scope-entity and the target. This milestone adds that arm without re-implementing reachability: the predicate calls `tree.ReachesScope`.

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

## Constraints

- Delegate to `tree.ReachesScope` — single source of truth for the three edges.
- Per M-0144's ADR (the `EvalContext` shape decision).
- `tdd: required`.

## Out of scope

The spec `Rule` that uses the predicate (M-0147); the cellcoverage fixtures (M-0146).

## Dependencies

M-0144 (ADR).

### AC-1 — EvaluatePredicate evaluates scope-reach for reachable and unreachable inputs

`EvaluatePredicate` evaluates a `scope-reach` predicate (no `unknown subject` error) for both reachable and unreachable inputs.

*Evidence:* a table test driving `EvaluatePredicate` over a fixture set with reachable / not-reachable cases.

### AC-2 — scope-reach predicate agrees with tree.ReachesScope across the fixture set

The predicate **agrees with `tree.ReachesScope`** — same verdict for the same (target, scope-entity) across the fixture set — proving no re-derivation of D-0006.

*Evidence:* a test asserting `EvaluatePredicate(scope-reach)` equals `tree.ReachesScope` for each fixture case.

### AC-3 — EvalContext carries the scope-entity and target the scope-reach arm needs

`EvalContext` carries the actor scope-entity + target the `scope-reach` arm needs (`Target` and `ScopeEntity` fields), with the shape documented in the `EvalContext` doc comment.

*Evidence:* the AC-1/AC-2 tests exercise the populated `EvalContext`; a branch-coverage audit of the new arm (reachable, unreachable, and the `cmpBool` op/value error paths).

