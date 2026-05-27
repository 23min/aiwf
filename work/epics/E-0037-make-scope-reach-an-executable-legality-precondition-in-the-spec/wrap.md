# Epic wrap — E-0037

**Date:** 2026-05-27
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0037-scope-reach-spec
**Merge commit:** feba2074

## Milestones delivered

All four merged to `main` via the epic integration merge `feba2074`:

- **M-0144** — ADR: represent a global precondition; classify out-of-scope as legality (ratifies ADR-0013)
- **M-0145** — Implement `scope-reach` in `EvaluatePredicate` with verb-invocation context
- **M-0146** — Extend cellcoverage with authorized-scope fixtures (`CellFixture.AuthorizeScope`)
- **M-0147** — Land global `scope-reach` rule; reclassify code; AC-5 fourth arm green

## Summary

E-0037 made `scope-reach` — D-0006's three-edge scope reachability — an executable, legality-classed predicate in the legal-workflow spec, completing the formal-model certification M-0141 deliberately deferred (G-0171). `EvaluatePredicate` now evaluates `scope-reach` by delegating to `tree.ReachesScope` (the runtime source of truth, no re-derivation); a marked global rule in `spec.GlobalRules()` carries the precondition; `provenance-authorization-out-of-scope` is reclassified to a `codes.ClassLegality` descriptor so the AC-5 fourth arm covers it; and the M-0146 authorized-scope cellcoverage machinery exercises it both ways. **No runtime reachability behavior changed** — the spec mirrors the M-0141 gate. Mid-flight, the global-rule representation pivoted from M-0144's ratified `Global`-flag-in-`Rules()` to a separate `GlobalRules()` accessor after implementation revealed the flag fanned skip-exceptions across ~6 meta-tests; ADR-0013 was amended to record the correction (the `Global` flag is now its rejected, originally-chosen alternative).

## ADRs ratified

- **ADR-0013** — Represent a global precondition; classify out-of-scope as legality (ratified at M-0144; amended at M-0147 to the separate-`GlobalRules()`-accessor mechanism).

## Decisions captured

- **D-0014** (E-0036) — the reconcile that split this arm out and set the directions (pre-existing, `accepted`).

## Follow-ups carried forward

- None specific to E-0037 — **G-0171 is closed** by this epic. The independent E-0036 follow-ups remain open and out of scope: **G-0169** (`--format=json` for non-`FinishVerb` verbs), **G-0170** (Apply-rollback data-loss — recurred as wrap friction here), and D-0005 / the `--evidence` gate (pending the philosophy walk-back).
- Non-tracked observation: `internal/check` carries two duplicate finding-code test helpers (`codes()` / `findingCodes()`); the M-0147 reclassification sidesteps the resulting import name-clash with an alias. Noted, not filed.

## Handoff

`scope-reach` is now spec-certified end to end — the verb-time legality rule that lived only in hand-written Go is inside the spec's bidirectional drift net. The authorized-scope cellcoverage machinery (`CellFixture.AuthorizeScope`, M-0146) is reusable for any future scope-gated rule. Nothing in this epic is deliberately left open.

## Doc findings

`wf-doc-lint` (scoped to the epic change-set): clean — ADR-0013's code references resolve, no doc-TODOs, no broken links. The epic's other changed files are Go + entity specs (not narrative `docs/` prose).
