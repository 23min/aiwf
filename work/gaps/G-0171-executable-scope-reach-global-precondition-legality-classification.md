---
id: G-0171
title: Executable scope-reach global precondition + legality classification
status: open
discovered_in: M-0141
---
## What's missing

D-0006's `scope-reach` reachability is, after M-0141, **enforced** at verb-time and check-time (both narrowed to the three-edge tree) and the out-of-scope refusal carries a structured code. What is *not* done:

1. **`scope-reach` is a documented-but-unimplemented spec predicate.** `internal/workflows/spec/spec.go` lists `scope-reach` in the closed `Predicate` Subject vocabulary (tied to D-0006), but `EvaluatePredicate` (`evaluate.go`) does not implement it — it returns `unknown subject`. The formal spec model therefore cannot *evaluate* a scope-reach precondition; only the hand-written Go gate (`verb.Allow`) can.

2. **The out-of-scope code is not legality-classed.** `provenance-authorization-out-of-scope` remains a `codes.ClassStructural` code (consistent with `codes.go` filing "provenance" under structural). Reclassifying it to `codes.ClassLegality` is inseparable from item 3 below, because the AC-5 fourth-arm chokepoint (`TestM0123_AC5_ImplToSpec_LegalityCodesReferenced`) requires every `ClassLegality` code to be the `ExpectedErrorCode` of an `OutcomeIllegal` spec `Rule`.

3. **No spec representation for a *global* precondition.** D-0006 says scope-reach is "a global precondition that applies to every cell rather than per-cell duplication," and explicitly defers the encoding: *"single global rule vs. per-cell precondition replication is settled during phase 1's schema concretization"* — which never happened. The `spec.Rule` table is keyed per-`(Kind, FromState, Verb)`; a cross-cutting precondition does not fit it, and the AC-5 fourth arm assumes per-cell `Rule` round-tripping. Representing and drift-certifying a global precondition is a spec-schema design question.

## Why it matters

The kernel's "principal × agent × scope" commitment is now *operationally* enforced (M-0141 closed the scope-leak bug), but the **formal legal-workflow spec** — E-0033's "verified source of truth for legal/illegal kernel workflows" — still cannot express or evaluate the scope-reach gate. Until it can, scope-reachability is the one verb-time legality rule that lives only in hand-written Go, outside the spec's bidirectional drift net. That is exactly the spec→impl drift class E-0036 exists to retire, for this one rule.

## Proposed shape (recommend its own epic)

This warrants an **ADR** and is plausibly its own small epic, not a tail milestone — per E-0036 open-question-1 (*"split ... if its verb-time-refusal design warrants its own ADR"*), which this does:

- **ADR:** is an out-of-scope refusal *legality* (verb-time precondition, named by an illegal cell) or *structural provenance* (an integrity finding), or both surfaced at two times? And: how is a **global** precondition represented in the spec table and drift-certified — a new rule class, a per-verb fan-out, or an extension to the AC-5 fourth arm that recognizes global preconditions?
- Implement `scope-reach` in `EvaluatePredicate`, threading the actor's active-scope-entity and target through `EvalContext` (verb-invocation context, not entity state — a different input shape than the existing four entity-side subjects).
- Extend the `internal/cellcoverage` fixture framework to stand up an authorized-scope context so the m0124/m0125 positive/negative drivers can exercise a scope-reach precondition.
- Reclassify `provenance-authorization-out-of-scope` to `codes.ClassLegality` and land the spec `Rule`(s) that name it, turning the AC-5 fourth arm green with the reclassified code.

## Relationship to M-0141

M-0141 ships the **behavior**: three-edge reachability at both enforcement sites, the structured `errors.As`-able out-of-scope code (shared verb-time + check-time), and the no-scope/out-of-reach split. This gap is the **formal-model certification** of that behavior — the part that is genuinely greenfield spec-schema design and should not be rushed into the end of E-0036. See D-0013-adjacent reconcile decision recorded during M-0141.
