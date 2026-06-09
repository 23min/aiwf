---
id: D-0018
title: branch-not-found subsumed by rung-pair-illegal; catalog cleanup defers to AC-9
status: proposed
---
## Context

M-0161/AC-2 (G-0201) shipped a single-rung-pair predicate at the verb-layer authorize carve-out, applied regardless of `BranchExists`. The new check at [`internal/verb/authorize.go`](../../internal/verb/authorize.go) reads `branchparse.RungOf(opts.CurrentBranch, opts.TrunkShort)` + `branchparse.RungOf(branchExplicit, opts.TrunkShort)` and refuses with `PreflightRungPairError` (code: `rung-pair-illegal`) when the pair is not in the legal set `{(trunk, epic), (epic, milestone), (milestone, patch), (epic, patch)}`.

The new predicate **subsumes** the prior `PreflightBranchNotFoundError` path:

- Pre-AC-2: a non-existent `--branch` value with non-ritual shape (e.g. `--branch garbage`) triggered the future-binding carve-out's "target must be ritual" check, returning `branch-not-found`.
- Post-AC-2: `RungOf("garbage", _)` returns `""` → `LegalRungPair(_, "")` is false → returns `rung-pair-illegal`.

The `PreflightBranchNotFoundError` type and `CodePreflightBranchNotFound` constant remain in [`internal/verb/authorize.go`](../../internal/verb/authorize.go) lines 48–110 but are no longer constructed by the verb. The pre-AC-2 `branch-not-found` code is now dead at its emission site.

Three downstream artifacts still claim this code is live:

- [`internal/workflows/spec/rules.go`](../../internal/workflows/spec/rules.go) lines 85–97 — `GlobalRules()` rule entry with `ExpectedErrorCode: "branch-not-found"`.
- [`internal/workflows/spec/branch/rules.go`](../../internal/workflows/spec/branch/rules.go) lines 40–49 — `branch-cell-2` with `ExpectedErrorCode: "branch-not-found"` and a comment naming `TestRunAuthorize_AITarget_BranchMissing_Refuses` as its test. The test was updated by AC-2 to assert `rung-pair-illegal`; the cell now lies about which code fires.
- [`internal/policies/m0158_ac2_corner_cells_test.go`](../../internal/policies/m0158_ac2_corner_cells_test.go) line 84 — keyword-set mapping `2: "branch-not-found"`.

Plus the embedded skill bodies (`aiwfx-start-epic`/`aiwfx-start-milestone` SKILL.md) still mention `branch-not-found` as a refusal code an operator might see — true for some paths still but slightly imprecise post-AC-2.

This decision records the choice of how to handle the spec-table / dead-code state, *flagged by the M-0161/AC-2 reviewer pass on 2026-06-04* (reviewer finding S-3).

## Decision

**Retain `PreflightBranchNotFoundError` and `CodePreflightBranchNotFound` as deprecated dead code; defer the spec-table cleanup to M-0161/AC-9 (G-0210 catalog refactor).**

Rationale:

1. **API stability.** The exported error type and code constant could in theory be matched on by downstream consumers (kernel-rule classification, programmatic error handling). Removing them in AC-2 — a verb-semantics change — would conflate two concerns: the predicate refinement (legitimate) and an exported-API removal (separate kernel-stability concern).

2. **AC-9 is the natural sweep point.** M-0161/AC-9 (closes G-0210) refactors the M-0158 spec catalog to mechanical-weight-only cells. The dropped cells include cases that no longer carry mechanical weight after rule changes. `branch-cell-2`'s stale `ExpectedErrorCode` is precisely this class — it's documentation-of-a-kernel-promise that has shifted. Sweeping it during AC-9 keeps the catalog change atomic.

3. **Cost of mid-cycle cleanup.** Retiring the error type and the spec-rule entry now would require: (a) deleting the type and code, (b) updating `GlobalRules()`, (c) updating `branch-cell-2`'s `ExpectedErrorCode` to match the new code, (d) updating the policy keyword mapping, (e) re-running every spec-policy test to catch transitive effects. That's a non-trivial change for negligible gain — AC-9 will rewrite the cells from scratch.

4. **Operator-facing impact: minimal.** The spec-table claim of `branch-not-found` is documentation, not a kernel guarantee. An operator reading the cell who then runs the prior fixture will see `rung-pair-illegal` instead — confusing but not blocking. The standing test infrastructure (`internal/policies/m0123_ac5_*` impl-to-spec drift checks) already passed at AC-2 commit time because the new rule entry covers the new code; the stale entry doesn't break anything mechanical.

## Concrete sequencing

- **Now (AC-2 wrap):** add this `D-NNN` to the milestone's `## Decisions made during implementation` section.
- **AC-9 cycle (catalog refactor):** the `branch-not-found` cell and its policy keyword-mapping are explicitly retired as part of the 9-cell drop M-0161/AC-9 enumerates. Per the AC-9 body table, `branch-cell-2` is a Legal-illegal cell (currently "illegal: branch-not-found"); AC-9's refactor either re-maps it to `rung-pair-illegal` (preserving the semantic with the new code) or drops it entirely if rung-pair-illegal's cell coverage subsumes it.
- **Future deprecation pass:** if a future release decides the verb-side type/code should be removed entirely, file a separate gap. Not in scope for E-0030.

## Why not the alternatives

- **Alternative A: remove dead code now.** Rejected per points 1, 3 above. Conflates concerns; AC-9 is the right sweep point.
- **Alternative B: keep the error path live by routing some refusals through `branch-not-found`.** Rejected because it re-introduces the pre-AC-2 dual-error-path complexity AC-2 explicitly subsumed. The single `rung-pair-illegal` is simpler and the AC-2 body locked the semantic.
- **Alternative C: silently let the drift sit without recording the decision.** Rejected because CLAUDE.md "Working with the user" requires honest contract framing — silently retaining a spec-table claim that doesn't match the verb is exactly the drift the spec-table methodology exists to catch.

## References

- M-0161/AC-2 (G-0201) commit `51462d52` — feat(authorize): ritual rung-pair predicate
- M-0161/AC-2 reviewer pass (subagent, 2026-06-04) — finding S-3
- M-0161/AC-9 (G-0210) body — catalog refactor (sweeps the stale cells)
- [`internal/verb/authorize.go`](../../internal/verb/authorize.go) lines 48–110 — dead error type
- [`internal/workflows/spec/rules.go`](../../internal/workflows/spec/rules.go) lines 85–97 — stale spec-rule entry
- [`internal/workflows/spec/branch/rules.go`](../../internal/workflows/spec/branch/rules.go) lines 40–49 — `branch-cell-2`
- [`internal/policies/m0158_ac2_corner_cells_test.go`](../../internal/policies/m0158_ac2_corner_cells_test.go) line 84 — keyword mapping
- ADR-0010 — branch-model decision the predicate enforces
