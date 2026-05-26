---
id: M-0147
title: Land global scope-reach rule; reclassify code; AC-5 fourth arm green
status: draft
parent: E-0037
depends_on:
    - M-0145
    - M-0146
tdd: required
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
