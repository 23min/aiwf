---
id: E-0037
title: Make scope-reach an executable legality precondition in the spec
status: done
---
## Goal

Make `scope-reach` (D-0006's three-edge scope reachability) an **executable, legality-classed predicate** in the legal-workflow spec, so the verb-time out-of-scope refusal lands inside the spec's bidirectional drift net — completing the formal-model certification M-0141 deliberately deferred.

## Context

M-0141 (E-0036) shipped the runtime behavior: `tree.ReachesScope` narrowed scope reachability to D-0006's exact three edges at **both** the verb-time gate (`verb.Allow`) and the check-time audit (`provenance-authorization-out-of-scope`), and the refusal now carries a structured `errors.As`-able code. But the formal model is not yet caught up:

- `scope-reach` is a **documented-but-unimplemented** spec predicate — `internal/workflows/spec/spec.go` lists it in the `Predicate` subject vocabulary, but `EvaluatePredicate` returns `unknown subject`.
- `provenance-authorization-out-of-scope` is `codes.ClassStructural`, so the AC-5 fourth arm imposes no spec-rule obligation on it.
- The legal-workflow spec (E-0033) therefore cannot express or evaluate the one verb-time legality rule that lives only in hand-written Go — outside the bidirectional drift net.

D-0014 split this arm out as **G-0171**, recommending its own epic per E-0036 open-question-1: the global-precondition spec schema is ADR-worthy greenfield that D-0006 explicitly deferred ("single global rule vs. per-cell replication ... settled during phase-1 schema concretization" — which never happened). **This epic changes no runtime reachability behavior** — `tree.ReachesScope` is the source of truth; the spec predicate mirrors it.

## Scope

In:

- An **ADR** resolving the global-precondition representation and the out-of-scope legality classification (open questions 1 + the classification).
- `scope-reach` implemented in `EvaluatePredicate`, threading the actor's active-scope-entity + target through `EvalContext`, **delegating to `tree.ReachesScope`** (no re-derivation of D-0006).
- A single **marked global / cross-cutting** spec `Rule` carrying the `scope-reach` precondition with `provenance-authorization-out-of-scope` as its `ExpectedErrorCode`.
- Reclassify `provenance-authorization-out-of-scope` → `codes.ClassLegality`; the AC-5 fourth-arm policy green with it included.
- Extend `internal/cellcoverage` to stand up an **authorized-scope context** so the `m0124`/`m0125` drivers exercise the global rule (sized in the ADR; documented fallback to a dedicated test only if the extension proves its own epic).

Out:

- Changing runtime reachability behavior — M-0141 owns it; this epic mirrors it into the spec.
- G-0169 (`--format=json` wiring for non-`FinishVerb` verbs) and G-0170 (Apply-rollback data-loss) — independent E-0036 follow-ups.
- D-0005 / the `--evidence` gate — carved out, pending the philosophy walk-back.

## Constraints

- **Single source of truth for the three edges:** the spec predicate AGREES with `tree.ReachesScope`, it does not re-derive D-0006.
- **No papering:** the global rule lands in the same coverage net as every other cell (full cellcoverage), not via an allowlist exemption — unless the ADR sizes the cellcoverage extension as its own epic and records the fallback explicitly. This epic exists *because* M-0141 refused to paper this in.
- No new spec-schema expressivity beyond what the global precondition needs (KISS); the global-rule marker is the minimal extension.
- **Reviewed reconcile:** read the real `spec` / AC-5 / `cellcoverage` code against the resolved direction before implementing each milestone; surface divergence before coding.
- AC promotion requires mechanical evidence — a Go test under `internal/policies/`, a kernel finding-rule, or a fixture-validation script — even on `tdd: none` milestones.

## Success criteria

- `scope-reach` is evaluable by `EvaluatePredicate` (no `unknown subject`) and delegates to `tree.ReachesScope`.
- `provenance-authorization-out-of-scope` is `codes.ClassLegality` and is the `ExpectedErrorCode` of the marked global spec rule; the AC-5 fourth-arm policy is green with the code included.
- The global rule is exercised by the cellcoverage drivers (positive: in-scope agent verb succeeds; negative: out-of-scope refused with the code) — or, if the ADR sizes the cellcoverage extension out, by a dedicated test with the exemption explicitly recorded.
- Every ADR listed in *ADRs produced* is merged; every decision listed in *Decisions* is `accepted`.

## Open questions (directions set during planning; the ADR formalizes)

1. **Global-precondition representation.** *Direction (Q2): a single marked global / cross-cutting `Rule` — a `KindAny` sentinel or a `Global` flag — keeping the `Rule` table the single source of truth for legality codes.* The ADR validates the exact mechanism against the `Rule` key-uniqueness + coverage meta-tests (`m0123_ac2/ac4`, `m0124/m0125`) and pins how the AC-5 fourth arm recognizes it.
2. **`EvalContext` shape.** `scope-reach` needs verb-invocation context (the actor's active-scope-entity + target) that the current entity-side `EvalContext` does not carry. *Resolution:* sized in the first implementation milestone; the predicate delegates to `tree.ReachesScope`.
3. **cellcoverage extension sizing.** *Direction (Q3): full integration — authorized-scope fixtures so the drivers exercise the global rule; fall back to a dedicated test + recorded exemption only if it proves its own epic.* Sized in the ADR.

## Risks

- The `cellcoverage` framework extension (standing up authorized-scope fixtures) is the unsized piece and may be the bulk of the epic. Mitigation: the ADR (M1) sizes it explicitly with a documented fallback before the implementation milestone commits.

## Milestones

Sequenced so the global rule lands **last**, atop a ready evaluator + cellcoverage support — no broken-CI intermediate (reclassifying or landing the rule before its consumers exist would turn the drivers or the AC-5 arm red).

- **M-0144 — ADR: represent a global precondition; classify out-of-scope as legality.** Keystone — resolves the representation mechanism, the legality classification, and the cellcoverage sizing. No deps.
- **M-0145 — Implement `scope-reach` in `EvaluatePredicate` with verb-invocation context.** Adds the evaluator arm, delegating to `tree.ReachesScope`. Depends M-0144.
- **M-0146 — Extend cellcoverage with authorized-scope fixtures.** The unsized-risk piece; sized by M-0144. Depends M-0144, M-0145.
- **M-0147 — Land global `scope-reach` rule; reclassify code; AC-5 fourth arm green.** Rule lands atop the ready evaluator + cellcoverage support; drivers exercise it. Closes G-0171. Depends M-0145, M-0146.

## Decisions

- **D-0014** (E-0036) — the reconcile that split this arm out and set the direction (`accepted`).
- New decisions land per milestone (the M1 ADR plus any in-flight choices).

## ADRs produced

- One ADR (M1) — global-precondition representation in the spec table + the out-of-scope legality classification.
