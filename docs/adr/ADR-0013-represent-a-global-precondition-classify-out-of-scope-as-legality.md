---
id: ADR-0013
title: Represent a global precondition; classify out-of-scope as legality
status: accepted
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

A global precondition is represented by a **`Global bool` field on `Rule`**, assembled as a dedicated `globalRules()` slice appended to the existing per-kind cell slices in `rules.go` — the same union-of-slices idiom the table is already built from, so the global rule lives *in* `Rules()` alongside the cells. The single `scope-reach` rule carries the full illegal-cell field set the `m0123_ac2` invariants require: `Global: true`, `Preconditions: []Predicate{{Subject: "scope-reach"}}`, `Outcome: OutcomeIllegal`, `RejectionLayer: RejectionLayerVerbTime`, `BlockingStrict: true`, `ExpectedErrorCode: "provenance-authorization-out-of-scope"`, and `Sources.Decision: "D-0006"`. Its `Kind` / `FromState` / `Verb` are left as zero values because no cell coordinate applies — the `Global` flag, not a coordinate, is what identifies it.

The design goal is that **the global rule lives in `Rules()` and satisfies every `m0123_ac2` / `m0123_ac4` / `m0123_ac5` invariant with no change to those tests**, the only special-casing being in the coverage drivers:

- **Key-uniqueness** (`m0123_ac2`, `TestM0123_AC2_KeyUnique`): the real uniqueness key is `(Kind, FromState, Verb, Outcome)` with the preconditions slice as the tiebreak for the legal-plus-preconditioned-illegal companion pattern — *not* the bare triple. The global rule's key `("", "", "", OutcomeIllegal)` is unique by construction (no real cell has an empty `Kind`), so it composes with no partition. The remaining per-cell invariants (`OutcomeNotUnspecified`, `IllegalImpliesRejectionLayer`, `VerbTimeImpliesBlockingStrict`, `IllegalImpliesErrorCode`) are met by the field set above; `EveryEntityFSMFromStateCovered` enumerates only the six real kinds, so the empty-`Kind` rule is transparent to it.
- **`LookupRules`** (`m0123_ac4`): `LookupRules` filters `Rules()` by the exact `(Kind, FromState, Verb)` triple, so a real lookup never returns the global rule (its coordinate is empty). The consequence is load-bearing: the `scope-reach` precondition is **not** evaluated through the per-cell `LookupRules` path — it is evaluated by a dedicated global-precondition arm that consults the `Global`-flagged rules on every applicable agent verb invocation, independent of cell lookup. The `m0123_ac4` invariants (`NoDuplicatesWithinResult`, `MatchesAllInputs`, the hit/miss cases) hold unchanged: they exercise real triples, and `LookupRules("", "", "")` is not among the FSM-enumerated inputs.
- **AC-5 fourth arm** (`m0123_ac5`, `TestM0123_AC5_ImplToSpec_LegalityCodesReferenced`): `specIllegalErrorCodes()` scans `Rules()` for every `OutcomeIllegal` `ExpectedErrorCode`; the global rule is in `Rules()`, so the reclassified legality code `provenance-authorization-out-of-scope` is referenced for free, with no change to the arm. This keeps the `Rule` table the single source of truth for legality codes.
- **Per-cell coverage drivers** (`m0124` positive / `m0125` negative): the *only* consumers that special-case the global rule. They construct a per-cell fixture from `(Kind, FromState, Verb)`, which an empty coordinate cannot satisfy, so they skip `Global` rules in the per-cell loop. The global rule is exercised instead by the authorized-scope driver sized below (M-0146), positive and negative.

The `Global` flag is honest about what `scope-reach` is: when `Global` is true, the reader knows the cell coordinate does not apply, and the flag — not a meaningless coordinate — is the identity the evaluation arm and the coverage drivers key off. It does not leak into kind-switching code, and it preserves the single-source-of-truth invariant without a parallel type. The rejected `KindAny` sentinel, separate-`GlobalRules()`-accessor, separate-type, and `RuleScope`-enum alternatives are recorded under *Alternatives considered*.

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
- The `Rule` struct gains one field. The `m0123_ac2` / `m0123_ac4` / `m0123_ac5` meta-tests are untouched — the global rule composes with them as-is; only the `m0124` / `m0125` coverage drivers special-case it (a skip in the per-cell loop, plus the authorized-scope path M-0146 adds). Blast radius is localized to the rule-table assembly and the drivers.
- The reclassified legality code enters the AC-5 drift net: from M-0147 forward, removing the global rule or the code's spec reference fails CI. The verb-time legality rule that lived only in hand-written Go is now inside the bidirectional model.
- The `Global` marker is the minimal schema addition; no broader spec-schema expressivity is added (KISS).

## Alternatives considered

- **`KindAny` sentinel on the `Kind` field.** Rejected. `scope-reach` is independent of all three key axes, not just `Kind`; a single `Kind` sentinel is honest on one axis while `FromState` and `Verb` carry meaningless values, and the sentinel risks leaking into per-kind iteration / `switch e.Kind` code that does not expect a non-kind value. Honesty and isolation both favor the flag.
- **A separate `Invariant` / `CrossCuttingRule` type.** Rejected as premature (YAGNI). `Rule` already carries every field a global precondition needs (`Preconditions`, `Outcome`, `ExpectedErrorCode`); a parallel type duplicates the struct and forces the AC-5 scanner to union two types — for exactly one global rule. Introduce the type on the third global rule, not the first.
- **A separate `GlobalRules()` accessor** (cells stay in `Rules()`; global rules in a parallel slice). Rejected after the reviewed reconcile weighed it against the real invariants. Its one advantage — the per-cell invariants and `m0124` / `m0125` drivers ignore the global rule with no skip branch — is mostly moot, because M-0146 extends those drivers to exercise the global rule regardless. Against it: the `m0123_ac5` fourth arm's `specIllegalErrorCodes()` scans `Rules()` only, so a separate accessor forces a change there to union `Rules()` ∪ `GlobalRules()`, weakening the "single `Rule` table is the source of truth for legality codes" goal the epic set. Keeping the global rule in `Rules()` leaves `ac2` / `ac4` / `ac5` untouched.
- **A `RuleScope` enum (`ScopeCell` / `ScopeGlobal`) instead of a `bool`.** Rejected for now (KISS). A binary distinction is a `bool`; the enum is the documented upgrade path if a third rule-scope shape ever appears.
- **Per-cell replication of the `scope-reach` precondition into every illegal cell.** Rejected. D-0006 explicitly deferred "single global rule vs. per-cell replication"; replication would duplicate the precondition across the entire grid, defeating the single-source-of-truth goal and making the drift net restate D-0006 in N places instead of one.

## References

- E-0037 (this epic); M-0144 (this milestone); M-0145, M-0146, M-0147 (implementation milestones)
- D-0006 (three-edge scope reachability), D-0011 (typed `Code` descriptor), D-0014 (narrow reachability; split formal-model arm)
- ADR-0011 (legal-workflow spec methodology), ADR-0012 (typed `Coded` error pattern)
- G-0171 (the split-out formal-model arm this epic closes)
- M-0141 (E-0036; shipped the runtime behavior this ADR mirrors into the spec)
