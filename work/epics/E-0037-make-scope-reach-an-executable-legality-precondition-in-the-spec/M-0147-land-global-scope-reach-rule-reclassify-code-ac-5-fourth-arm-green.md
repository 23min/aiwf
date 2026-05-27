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
      status: met
      tdd_phase: done
    - id: AC-2
      title: provenance-authorization-out-of-scope is ClassLegality; AC-5 fourth arm green
      status: met
      tdd_phase: done
    - id: AC-3
      title: Cellcoverage machinery exercises the global rule positive and negative
      status: met
      tdd_phase: done
---
## Goal

Land the marked **global `scope-reach` rule** in the spec, **reclassify `provenance-authorization-out-of-scope` to `codes.ClassLegality`**, turn the **AC-5 fourth arm green** with the code included, and have the **cellcoverage drivers exercise the rule**. With the evaluator (M-0145) and cellcoverage support (M-0146) already in place, the rule lands last so every consumer is ready — no broken-CI intermediate.

## Context

This is the milestone that closes G-0171: the verb-time out-of-scope refusal becomes a first-class, evaluable, legality-classed spec rule inside the bidirectional drift net. The reclassification and the rule must land together (reclassifying alone turns the AC-5 fourth arm red until a rule names the code).

## Acceptance criteria

(ACs allocated separately via `aiwf add ac` after milestone creation; bodies seeded at allocation time.)

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

The global rule is exercised both ways through the M-0146 authorized-scope machinery (positive: in-scope agent verb succeeds; negative: out-of-scope refused with the rule's `ExpectedErrorCode`). It lives in `spec.GlobalRules()`, not `Rules()`, so the per-cell `m0124`/`m0125` drivers never see it — full integration per M-0144, not the recorded fallback.

*Evidence:* a test reading the global rule's code from `spec.GlobalRules()` and asserting the authorized-scope machinery refuses out-of-scope with exactly that code.

## Work log

### AC-1 — Global scope-reach rule present; key-uniqueness and coverage meta-tests green
Landed the `scope-reach` rule in `spec.GlobalRules()`; shape asserted (Illegal / VerbTime / BlockingStrict / `scope-reach == false` / `D-0006`); the `m0123` meta-tests stay green · commit `8d271be0` · `TestM0147_AC1_GlobalRulePresent`.

### AC-2 — provenance-authorization-out-of-scope is ClassLegality; AC-5 fourth arm green
Reclassified the code to a typed `codes.Code{…, ClassLegality}` descriptor; `collectImplFindingCodes` classifies it legality and the fourth arm (`TestM0123_AC5_ImplToSpec_LegalityCodesReferenced`) is green with the code referenced by the global rule · commit `18738f14` · `TestM0147_AC2_CodeIsLegality`.

### AC-3 — Cellcoverage machinery exercises the global rule positive and negative
Spec↔runtime tie: a test reads the global rule's `ExpectedErrorCode` from `spec.GlobalRules()` and asserts the M-0146 authorized-scope machinery refuses out-of-scope with exactly that code (in-scope succeeds, no commit on refusal) · commit `cdc70ca1` · `TestM0147_AC3_GlobalRuleExercised`.

## Decisions made during implementation

- **Pivot from the `Global`-flag-in-`Rules()` mechanism to a separate `GlobalRules()` accessor — amends ADR-0013.** M-0144 ratified the `Global` flag; implementation revealed it required a `Global` skip at ~6 cell-consuming meta-tests (`m0124`/`m0125`, `KindsResolve`, `VerbsResolve`, `FixtureSatisfiesIllegalPreconditions`, `IllegalCellsAllCovered`) — every `Rules()`-iterating check that assumes a cell coordinate. Scattered skips are fragile (a future check can forget one), so the decision was corrected to the separate accessor: zero skips, per-cell consumers untouched, only the two code-oriented AC-5 arms union `Rules() ∪ GlobalRules()`. ADR-0013's Decision subsection + Alternatives were amended in this milestone (with an M-0147 amendment note), and M-0144's `adr_0013_test.go` updated to match.
- **Reclassification graduates the bare-string const to a typed `codes.Code` descriptor** (D-0011 pattern, as M-0139 did for the cancel guards): `check.CodeProvenanceAuthorizationOutOfScope` is now `codes.Code{ID: …, Class: codes.ClassLegality}`; consumers (`verb/scope_errors.go`, `check/provenance.go`, the check tests) read `.ID`. The `codes` import is aliased (`codespkg`) in `provenance.go` to avoid the `check` package's top-level `codes()` test helper.

## Validation

- `go build ./...` — clean. `go test ./...` — 56 packages ok, 0 non-flake failures. `aiwf check` — 0 errors.
- M-0147 tests green (AC-1/AC-2/AC-3). The meta-tests that the in-`Rules()` attempt broke (`SpecRuleStructShape`, the AC-5 `KindsResolve`/`VerbsResolve` arms, `FixtureSatisfiesIllegalPreconditions`, `IllegalCellsAllCovered`) are green **with no skips** after the pivot. ADR-0013's structural test green against the amended body.
- TDD: AC-1 RED (`Rule.Global` undefined / rule absent) → GREEN; AC-2/AC-3 characterize the landed rule + machinery. All `met` at `phase: done`.
- Branch coverage: `GlobalRules()` returns a literal (no branches); the descriptor has none; the AC-5 unions are test-side. Nothing to audit.

## Deferrals

None. Closes **G-0171**. (Unrelated pre-existing debt left untouched: `internal/check` carries two duplicate finding-code test helpers, `codes()` and `findingCodes()`; the reclassification's import alias sidesteps the resulting name clash. Noted, not in scope.)

## Reviewer notes

- **The separate-accessor design was the operator's call after a rigor challenge** ("skips sound like papering"). The originally-ratified `Global` flag fanned skips across every `Rules()`-consumer; the accessor expresses "not a cell" once, structurally. See ADR-0013's amendment note for the full rationale.
- **AC-3 is the spec↔runtime tie**: it reads the global rule's `ExpectedErrorCode` from `spec.GlobalRules()` and asserts the runtime refuses out-of-scope with exactly that code — proving the spec mirror and the M-0141 gate agree, which is the epic's whole point.
- **The reclassification blast radius included `_test.go` files** (`provenance_test.go`'s 8 `.ID` sites + the `codes()` helper name collision) — missed by an initial non-test grep, caught by the build. Lesson: code-graduation greps must include test files.
- **G-0170 friction recurred**: the AC-2 `--phase done` promote flaked on `m0125` in its pre-commit and rolled back; retried clean.

