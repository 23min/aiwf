---
id: D-0014
title: Narrow scope reachability to D-0006 three edges; split formal-model arm
status: accepted
---
## Context

M-0141 set out to "implement D-0006 three-edge scope reachability" on the premise (from the milestone spec) that this was greenfield — that "no reachability check exists" and `internal/scope` was FSM-only. The mandated reviewed-reconcile read of the impl against D-0006 found that premise false:

- **Reachability is already enforced at two sites.** Verb-time: `verb.Allow` → `scopeAllowsAct` → `tree.Reaches` (`internal/verb/allow.go`), wired through `gateAndDecorate`. Check-time: `internal/check/provenance.go` fires `provenance-authorization-out-of-scope` via the same `tree.Reaches`. The verb-layer enforcement hook the milestone flagged as the "unsized risk" already exists.
- **Both use the full reference graph.** `tree.Reaches` (`internal/tree/tree.go`) walks all of `entity.ForwardRefs` — parent, depends_on, addressed_by, relates_to, supersedes, discovered_in, linked_adrs. D-0006 mandates exactly three edges (parent-forward, composite-containment, `discovered_in`-reverse) and **explicitly excludes** the governance edges.
- **The divergence is over-permissiveness, not a gap.** D-0006's `discovered_in` friction case already passes today (discovered_in is a forward edge). The real bug: an agent scoped on `E-1` can reach a cross-epic milestone via `depends_on`, or an ADR via a `supersedes`/`linked_adrs` chain, and the verb is wrongly allowed — scope boundaries leak through governance edges.
- **The verb-time refusal carries no structured code** and conflates "no active scope" with "scope exists but target out of reach."

`tree.Reaches`/`ReachesAny` have exactly two production callers, both scope-reachability (verb.Allow, check/provenance.go); no other consumer depends on the broad walk.

## Resolution

1. **Narrow reachability to D-0006's three edges, expressed explicitly.** Replace the full-graph `tree.Reaches` with a scope-reachability function that traverses only: the target's `parent` chain to the scope-entity; composite-id rollup (`M/AC-N` → `M`); and a single `discovered_in` hop followed by parent-climb. No governance edge is reachable *by construction* — the closed-set design D-0006 demands ("new kinds need explicit edge participation, never implicit graph traversal") becomes literally true in code. Both `verb.Allow` and `check/provenance.go` repoint to it; the dead full-graph walk is removed.

2. **One structured code, shared across both enforcement sites.** The verb-time refusal emits an `errors.As`-able code via M-0138's `Coded` pattern, reusing the existing `provenance-authorization-out-of-scope` string the check finding already uses — one code per violation, surfaced at two times (the same unification logic as M-0143's C2). Split the verb-time path so "scope exists but target out of reach" emits this code, distinct from "no active scope" (which keeps `provenance-no-active-scope`).

3. **Keep the code `codes.ClassStructural` for now.** `codes.go` files "provenance" under `ClassStructural`, and the check finding has always been structural. Reclassifying to `ClassLegality` is inseparable from the spec-model work (the AC-5 fourth arm requires every legality code to round-trip through an `OutcomeIllegal` spec `Rule`), so it travels with that work — not this milestone.

4. **Split the formal-model arm out (G-0171, recommend its own epic).** Making `scope-reach` an *executable* spec predicate (implement it in `EvaluatePredicate` with verb-invocation context in `EvalContext`; extend `cellcoverage`; reclassify to legality; represent and drift-certify a *global* precondition that does not fit the per-`(Kind,FromState,Verb)` `Rule` table) is genuinely greenfield spec-schema design — D-0006 deferred the encoding and never settled it. Per E-0036 open-question-1, a verb-time-refusal design that warrants its own ADR is split; this does. M-0141 keeps the behavior fix (which is the substance of G-0143); the certification arm becomes G-0171.

## Alternatives considered

- **Keep the full-graph walk:** rejected — it is the scope-leak bug; governance edges punch through scope boundaries, violating scope-as-governance semantics (D-0006).
- **Narrow only the verb-time gate, leave check-time on the full graph:** rejected — two sources of truth for one rule; check-time exists to audit exactly what the verb-time gate enforces (the impl comment already says "the same rule"). They must share one predicate.
- **Do the legality classification + spec-model integration now (in M-0141):** rejected — the global-precondition schema is undecided (D-0006 deferred it), unsized, and would force either a hollow/contradictory `Rule` cell or `cellcoverage` framework surgery under epic-tail pressure. Both are hacks; splitting is the purer engineering choice, not the lazier one.

## Consequences

- M-0141 closes G-0143 (three-edge reachability + verb-time out-of-scope refusal with a structured code) and is self-contained with clean mechanical evidence.
- The check-time rule becomes stricter; M-0141 must verify the narrowed rule does not newly-flag this repo's own authorized history before wrapping (surface, don't paper over, if it does).
- The `scope-reach` predicate remains documented-but-unimplemented in the spec until G-0171; the out-of-scope code stays structural until then.
- E-0036's headline goal (drain `deferredImplErrorCodes`, now only `ac-evidence-missing` / D-0005, carved out) is unaffected — M-0141 was the reviewed-reconcile add-on, and its substantive deliverable lands in full.
