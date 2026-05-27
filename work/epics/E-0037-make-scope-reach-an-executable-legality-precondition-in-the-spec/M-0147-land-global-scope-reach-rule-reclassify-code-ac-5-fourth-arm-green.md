---
id: M-0147
title: Land global scope-reach rule; reclassify code; AC-5 fourth arm green
status: in_progress
parent: E-0037
depends_on:
    - M-0145
    - M-0146
tdd: required
acs:
    - id: AC-1
      title: Global scope-reach rule present; key-uniqueness and coverage meta-tests green
      status: open
      tdd_phase: done
    - id: AC-2
      title: provenance-authorization-out-of-scope is ClassLegality; AC-5 fourth arm green
      status: open
      tdd_phase: green
    - id: AC-3
      title: Cellcoverage machinery exercises the global rule positive and negative
      status: open
      tdd_phase: green
---
## Goal

Land the marked **global `scope-reach` rule** in the spec, **reclassify `provenance-authorization-out-of-scope` to `codes.ClassLegality`**, turn the **AC-5 fourth arm green** with the code included, and have the **cellcoverage drivers exercise the rule**. With the evaluator (M-0145) and cellcoverage support (M-0146) already in place, the rule lands last so every consumer is ready — no broken-CI intermediate.

## Context

This is the milestone that closes G-0171: the verb-time out-of-scope refusal becomes a first-class, evaluable, legality-classed spec rule inside the bidirectional drift net. The reclassification and the rule must land together (reclassifying alone turns the AC-5 fourth arm red until a rule names the code).

## Acceptance criteria

- **AC1** — The marked global rule exists per M-0144's mechanism (`Outcome: Illegal`, `ExpectedErrorCode: provenance-authorization-out-of-scope`, the `scope-reach` precondition); the `Rule` key-uniqueness + coverage meta-tests stay green. *Evidence:* a spec assertion the rule is present + the existing meta-tests green.
- **AC2** — `provenance-authorization-out-of-scope` is `codes.ClassLegality` and the AC-5 fourth arm (`TestM0123_AC5_ImplToSpec_LegalityCodesReferenced`) is green with the code included. *Evidence:* the legality-class scan includes the code + the AC-5 fourth-arm policy passes.
- **AC3** — The cellcoverage drivers exercise the global rule (positive: in-scope agent verb succeeds; negative: out-of-scope refused with the code), or — per M-0144's recorded fallback — a dedicated test does, with the AC-4 exemption documented. *Evidence:* `m0124`/`m0125` coverage of the global rule (or the recorded-fallback test).

## Constraints

- Rule lands atop the ready evaluator (M-0145) + cellcoverage support (M-0146) — no broken-CI intermediate state.
- **No papering** — the rule joins the same coverage net as every other cell unless M-0144 sized the fallback explicitly.
- `tdd: required`.

## Out of scope

Any change to runtime reachability (`tree.ReachesScope`) — M-0141 owns it; this milestone mirrors it into the spec.

## Dependencies

M-0145 (evaluator), M-0146 (cellcoverage support). Closes **G-0171**.

### AC-1 — Global scope-reach rule present; key-uniqueness and coverage meta-tests green

The marked global rule exists per M-0144's mechanism (`Global: true`, `Outcome: Illegal`, `ExpectedErrorCode: provenance-authorization-out-of-scope`, the `scope-reach` precondition); the `Rule` key-uniqueness + coverage meta-tests stay green.

*Evidence:* a spec assertion the rule is present with the right shape + the existing `m0123` meta-tests green.

### AC-2 — provenance-authorization-out-of-scope is ClassLegality; AC-5 fourth arm green

`provenance-authorization-out-of-scope` is `codes.ClassLegality` (a typed `codes.Code` descriptor per D-0011) and the AC-5 fourth arm (`TestM0123_AC5_ImplToSpec_LegalityCodesReferenced`) is green with the code included.

*Evidence:* the legality-class scan (`collectImplFindingCodes`) classifies the code `ClassLegality` + the AC-5 fourth-arm policy passes.

### AC-3 — Cellcoverage machinery exercises the global rule positive and negative

The global rule is exercised both ways through the M-0146 authorized-scope machinery (positive: in-scope agent verb succeeds; negative: out-of-scope refused with the rule's `ExpectedErrorCode`), and the per-cell `m0124`/`m0125` drivers skip it (no cell coordinate) — full integration per M-0144, not the recorded fallback.

*Evidence:* a test reading the global rule's code from `spec.Rules()` and asserting the authorized-scope machinery refuses out-of-scope with exactly that code; the `m0124`/`m0125` skip keeps them green.

