---
id: M-0145
title: Implement scope-reach in EvaluatePredicate with verb-invocation context
status: done
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

## Work log

### AC-1 — EvaluatePredicate evaluates scope-reach for reachable and unreachable inputs
`scope-reach` arm added to `EvaluatePredicate`; table test over D-0006's edges (reachable + unreachable) passes with no unknown-subject error · commit `48b8599d` · `TestEvaluatePredicate_ScopeReach` green.

### AC-2 — scope-reach predicate agrees with tree.ReachesScope across the fixture set
Predicate verdict asserted equal to `tree.ReachesScope` for every fixture case — delegation, not re-derivation of D-0006 · commit `3d786da8`.

### AC-3 — EvalContext carries the scope-entity and target the scope-reach arm needs
`EvalContext` widened with `Target` + `ScopeEntity`; `cmpBool` branch coverage (==/!=, true/false, error paths) · commit `ca9738c8` · `TestEvaluatePredicate_ScopeReach_OpContract` green.

## Decisions made during implementation

- **`EvalContext` shape (the decision ADR-0013 deferred here).** M-0144's ADR deferred the `EvalContext` shape to "the first implementation milestone." Decided: add explicit `Target string` + `ScopeEntity string` (verb-invocation context); the arm is `cmpBool(p.Op, t.ReachesScope(ctx.Target, ctx.ScopeEntity), p.Value)`. Chosen over a `ScopeEntity`-only shape (target = `e.ID`) because `scope-reach` is a verb-invocation predicate, AC-3 specifies the context carries both, and move/create targets aren't the evaluated entity `e`. M-0146 populates these fields; M-0147's rule consumes the predicate.
- **Predicate Op / polarity contract.** The predicate returns reachability (true when reachable), matching `tree.ReachesScope` (AC-2). A strict `cmpBool` supports `==`/`!=` against `"true"`/`"false"` so M-0147's illegal rule can express the out-of-scope violation as `scope-reach == false` (the predicate-true-when-violation convention every other illegal cell follows). A bare/unary form was rejected — it could not express that polarity.

## Validation

- `go test ./internal/workflows/spec/` — green (`TestEvaluatePredicate_ScopeReach` + `_OpContract`, 8 subtests). `go test ./internal/policies/` — green. `go build ./...` — clean. `aiwf check` — 0 errors.
- TDD: RED confirmed first (`unknown field Target in EvalContext`) → GREEN; all three ACs `met` at `phase: done`.
- Branch coverage: `cmpBool` 100%; the `scope-reach` arm exercised reachable + unreachable. The single uncovered block (`evaluate.go` `self.tdd_phase` `AC == nil` guard) is pre-existing, outside this diff.

## Deferrals

None. The spec `Rule` that uses the predicate (M-0147) and the cellcoverage authorized-scope fixtures that populate `EvalContext.Target` / `ScopeEntity` (M-0146) are sequenced follow-ons, not deferred scope.

## Reviewer notes

- **Delegation, not re-derivation.** The arm is one line calling `tree.ReachesScope`; AC-2 mechanically asserts agreement across the fixture set, so the spec predicate cannot silently drift from D-0006.
- **`EvaluatePredicate` is spec-side, not the runtime gate.** It is exercised by the cellcoverage drivers + tests; the runtime gate remains `verb/allow.go`. This milestone adds the spec mirror — it changes no runtime reachability behavior (per the epic constraint).
- **Polarity is M-0147's to wire.** The predicate returns reachability; the illegal global rule will read `scope-reach == false`. `cmpBool` exists to carry that polarity.
- **Flake (G-0097), investigated not papered.** `TestM0124_PositiveDriver` flaked once in a full-suite run (parallel git-subprocess contention); confirmed not introduced here — passes in isolation, the comprehensive predicate-vocab test is unchanged, and it vanished on re-run.

