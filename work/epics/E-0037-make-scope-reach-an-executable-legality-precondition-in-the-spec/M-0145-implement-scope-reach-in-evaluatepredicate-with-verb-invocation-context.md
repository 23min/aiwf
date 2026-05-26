---
id: M-0145
title: Implement scope-reach in EvaluatePredicate with verb-invocation context
status: draft
parent: E-0037
depends_on:
    - M-0144
tdd: required
---
## Goal

Implement the `scope-reach` subject in `EvaluatePredicate`, threading the actor's active-scope-entity + target through `EvalContext`, **delegating to `tree.ReachesScope`** (the M-0141 source of truth) rather than re-deriving D-0006.

## Context

`scope-reach` is in the spec's `Predicate` subject vocabulary (`spec.go`) but `EvaluatePredicate` (`evaluate.go`) returns `unknown subject` for it. The other four subjects are entity-side (`self.`/`parent.`/`all-children.`/`any-child.`); `scope-reach` is the first that needs *verb-invocation* context — the actor's active scope-entity and the target. This milestone adds that arm without re-implementing reachability: the predicate calls `tree.ReachesScope`.

## Acceptance criteria

- **AC1** — `EvaluatePredicate` evaluates a `scope-reach` predicate (no `unknown subject` error) for both reachable and unreachable inputs. *Evidence:* a table test driving `EvaluatePredicate` over a fixture set with reachable / not-reachable cases.
- **AC2** — The predicate **agrees with `tree.ReachesScope`** — same verdict for the same (target, scope-entity) across the fixture set — proving no re-derivation. *Evidence:* a test asserting `EvaluatePredicate(scope-reach)` equals `tree.ReachesScope` for each fixture case.
- **AC3** — `EvalContext` carries the actor scope-entity + target needed by the arm, with the shape documented. *Evidence:* the AC1/AC2 tests exercise the populated `EvalContext`; a branch-coverage audit of the new arm.

## Constraints

- Delegate to `tree.ReachesScope` — single source of truth for the three edges.
- Per M-0144's ADR (the `EvalContext` shape decision).
- `tdd: required`.

## Out of scope

The spec `Rule` that uses the predicate (M-0147); the cellcoverage fixtures (M-0146).

## Dependencies

M-0144 (ADR).
