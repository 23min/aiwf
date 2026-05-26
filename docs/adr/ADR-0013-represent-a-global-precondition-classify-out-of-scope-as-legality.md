---
id: ADR-0013
title: Represent a global precondition; classify out-of-scope as legality
status: proposed
---
## Context

E-0037 makes `scope-reach` — D-0006's three-edge scope reachability — an executable, legality-classed predicate in the legal-workflow spec (ADR-0011), so the verb-time out-of-scope refusal lands inside the spec's bidirectional drift net. M-0141 (E-0036) shipped the runtime behavior: `tree.ReachesScope` is the single source of truth for the three edges, enforced at both the verb-time gate (`verb/allow.go`) and the check-time audit (`check/provenance.go`). The formal model is not yet caught up.

Three facts from the reviewed reconcile against the real code constrain this decision:

1. **The spec `Rule` table is a per-cell legality grid.** Every `Rule` is keyed by its `(Kind, FromState, Verb)` triple (`spec.go`), and `rules.go` assembles the table by concatenating per-kind rule slices. A `Rule`'s identity *is* its cell coordinate. There is no global / cross-cutting rule mechanism today.
2. **`scope-reach` is not a cell.** It is a cross-cutting precondition — an authorized agent may act only within its scope — that applies to *every* act/move/create verb regardless of kind, from-state, or verb. It is orthogonal to all three key axes. The predicate is listed in the spec vocabulary (`spec.go`) but `EvaluatePredicate` has no arm for it and returns `unknown subject` (`evaluate.go`).
3. **`provenance-authorization-out-of-scope` is currently `codes.ClassStructural`.** It is a bare string constant (`check/provenance.go`), emitted at two surfaces — the verb-time refusal and the check-time audit. The AC-5 fourth arm (`m0123_ac5_drift_test.go`) requires every `codes.ClassLegality` code to be named by ≥1 illegal-outcome spec `Rule`; an unreferenced legality code fails the drift net.

This ADR resolves how a global precondition is represented in the `Rule` table, classifies out-of-scope as legality, and sizes the cellcoverage extension the implementation milestones (M-0145, M-0146, M-0147) depend on. It changes no runtime reachability behavior — `tree.ReachesScope` remains the source of truth; the spec predicate mirrors it.

## Decision

### Global-rule representation

A global precondition is represented by a **`Global bool` field on `Rule`**, assembled as a dedicated `globalRules()` slice appended to the existing per-kind cell slices in `rules.go` — the same union-of-slices idiom the table is already built from. The single `scope-reach` rule carries `Global: true`, `Preconditions: []Predicate{{Subject: "scope-reach"}}`, `Outcome: OutcomeIllegal`, and `ExpectedErrorCode: "provenance-authorization-out-of-scope"`. Its `Kind` / `FromState` / `Verb` are left as zero values because no cell coordinate applies.

Composition with the existing meta-tests:

- **Key-uniqueness** (`m0123_ac2`): keys `(Kind, FromState, Verb)` over **non-`Global`** rules only. Global rules carry no cell coordinate, so they are excluded from cell-keying; the invariant becomes "unique among cells," which is exactly true. The check partitions on the flag.
- **Per-cell coverage drivers** (`m0124` positive / `m0125` negative): iterate **non-`Global`** cells. The global rule is exercised separately by the authorized-scope driver sized below (M-0146), not by the per-cell grid.
- **AC-5 fourth arm** (`m0123_ac5`): scans **all** rules — cells ∪ global — for legality-code coverage. A `Global` illegal rule naming `provenance-authorization-out-of-scope` satisfies the arm's requirement that the reclassified legality code be referenced. This keeps the `Rule` table the single source of truth for legality codes.

The `Global` flag is honest about what `scope-reach` is: when `Global` is true, the reader knows the cell coordinate does not apply. It does not leak into kind-switching code, and it preserves the single-source-of-truth invariant without a parallel type. The rejected `KindAny` sentinel, separate-type, and `RuleScope`-enum alternatives are recorded under *Alternatives considered*.

### Out-of-scope classification as legality

`provenance-authorization-out-of-scope` is reclassified from `codes.ClassStructural` to **`codes.ClassLegality`**, promoted from a bare string constant to a typed `Code{Class: codes.ClassLegality}` descriptor per ADR-0012 / D-0011.

The code is **dual-emitted**: the verb-time gate (`verb/allow.go`, via the scope-out-of-reach refusal) and the check-time audit (`check/provenance.go`) raise the *same* code for the *same* violation at two surfaces. This is one legality violation observed twice, not two distinct findings — the classification names the violation, and both emission sites agree on it. This mirrors every other legality code in the spec, which the runtime refuses at verb time and the audit re-detects at check time.

Carve-out note: reclassifying the code to `codes.ClassLegality` arms the AC-5 fourth arm's obligation that the code be named by an illegal-outcome spec `Rule`. That obligation is satisfied by — and only by — the global rule above. The reclassification and the global rule therefore land together in M-0147; reclassifying before the rule exists would turn the fourth arm red, which is why the epic sequences the rule last.

### cellcoverage extension sizing

The cellcoverage drivers carry **no authorized-scope scaffolding** today: `CellFixture` (`internal/cellcoverage`) drives every cell with a single `human/test` actor and has no `authorize` / scope machinery. Exercising the global `scope-reach` rule requires a fixture that stands up an active authorization scope and runs a verb as an in-scope vs out-of-scope `ai/<id>` agent.

Sizing: this is **tractable as full integration within M-0146**, not a new framework. `CellFixture` already performs in-process verb setup; the increment is (1) an in-process `authorize` opener seeding an active scope commit, (2) the scope-entity in the fixture tree, (3) threading the `ai/<id>` actor + scope into the verb call, and (4) consuming the `EvalContext` scope fields that M-0145 adds. Each step is additive to the existing fixture; none rewrites the driver model.

Explicit fallback: **if** the `EvalContext` threading or the authorized-scope fixture scaffolding proves to exceed a single milestone's worth of work — i.e. M-0146 cannot land it without itself becoming an epic — **then** the global rule is exercised by a dedicated authorized-scope test under `internal/policies/`, and the cellcoverage exemption is recorded explicitly (the global rule named in an allowlist with this ADR as the rationale). The fallback is a documented escape hatch, not the plan; full integration is the plan.

## Consequences

- `EvaluatePredicate` gains a `scope-reach` arm (M-0145) that delegates to `tree.ReachesScope` — no re-derivation of D-0006's edges; the spec agrees with the runtime, it does not restate it.
- `EvalContext` widens to carry the actor's active-scope entity and the target (M-0145), the context `scope-reach` needs and the current entity-side context lacks.
- The `Rule` struct gains one field; the key-uniqueness and coverage meta-tests gain a partition on it. Blast radius is localized to the rule-table assembly and those tests.
- The reclassified legality code enters the AC-5 drift net: from M-0147 forward, removing the global rule or the code's spec reference fails CI. The verb-time legality rule that lived only in hand-written Go is now inside the bidirectional model.
- The `Global` marker is the minimal schema addition; no broader spec-schema expressivity is added (KISS).

## Alternatives considered

- **`KindAny` sentinel on the `Kind` field.** Rejected. `scope-reach` is independent of all three key axes, not just `Kind`; a single `Kind` sentinel is honest on one axis while `FromState` and `Verb` carry meaningless values, and the sentinel risks leaking into per-kind iteration / `switch e.Kind` code that does not expect a non-kind value. Honesty and isolation both favor the flag.
- **A separate `Invariant` / `CrossCuttingRule` type.** Rejected as premature (YAGNI). `Rule` already carries every field a global precondition needs (`Preconditions`, `Outcome`, `ExpectedErrorCode`); a parallel type duplicates the struct and forces the AC-5 scanner to union two types — for exactly one global rule. Introduce the type on the third global rule, not the first.
- **A `RuleScope` enum (`ScopeCell` / `ScopeGlobal`) instead of a `bool`.** Rejected for now (KISS). A binary distinction is a `bool`; the enum is the documented upgrade path if a third rule-scope shape ever appears.
- **Per-cell replication of the `scope-reach` precondition into every illegal cell.** Rejected. D-0006 explicitly deferred "single global rule vs. per-cell replication"; replication would duplicate the precondition across the entire grid, defeating the single-source-of-truth goal and making the drift net restate D-0006 in N places instead of one.

## References

- E-0037 (this epic); M-0144 (this milestone); M-0145, M-0146, M-0147 (implementation milestones)
- D-0006 (three-edge scope reachability), D-0011 (typed `Code` descriptor), D-0014 (narrow reachability; split formal-model arm)
- ADR-0011 (legal-workflow spec methodology), ADR-0012 (typed `Coded` error pattern)
- G-0171 (the split-out formal-model arm this epic closes)
- M-0141 (E-0036; shipped the runtime behavior this ADR mirrors into the spec)
