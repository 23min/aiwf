---
id: G-0417
title: Dead branch-not-found code and stale rung-pair-illegal spec-table entries
status: open
discovered_in: M-0161
---
## What's missing

A mechanical sweep of the dead `branch-not-found` surface, orphaned when its
originally-named cleanup point (M-0161/AC-9) was cancelled:

- **Dead code**: `PreflightBranchNotFoundError` and `CodePreflightBranchNotFound`
  in [`internal/verb/authorize.go`](../../internal/verb/authorize.go) (lines
  ~48-124) are no longer constructed anywhere — M-0161/AC-2's rung-pair
  predicate subsumed the path that used to return them.
- **Stale spec-table entries** still citing the retired code instead of its
  replacement (`rung-pair-illegal`):
  - [`internal/workflows/spec/rules.go`](../../internal/workflows/spec/rules.go)
    `GlobalRules()` — `ExpectedErrorCode: "branch-not-found"`.
  - [`internal/workflows/spec/branch/rules.go`](../../internal/workflows/spec/branch/rules.go)
    `branch-cell-2` — `ExpectedErrorCode: "branch-not-found"`.
  - [`internal/policies/m0158_ac2_corner_cells_test.go`](../../internal/policies/m0158_ac2_corner_cells_test.go)
    line 89 — keyword-set mapping `2: "branch-not-found"`.

## Why it matters

[D-0018](../decisions/D-0018-branch-not-found-subsumed-by-rung-pair-illegal-catalog-cleanup-defers-to-ac-9.md)
deliberately retained this as deprecated dead code and named M-0161/AC-9 as
the natural sweep point. AC-9 was cancelled (per
[D-0022](../decisions/D-0022-m-0161-ac-9-deferred-to-follow-up-milestone-m-0161-wraps-8-9.md))
and its scope moved to M-0162, which shipped the full bijection/Pin-registry
catalog refactor and closed the parent gap
([G-0210](../gaps/archive/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md))
— but G-0210's own target catalog (written before this finding existed)
explicitly kept `branch-cell-2` labeled `branch-not-found` verbatim. A
sibling gap ([G-0224](../gaps/archive/G-0224-aiwfx-start-epic-start-milestone-skill-md-cites-retired-branch-not-found-code.md))
fixed the two SKILL.md mentions of the stale code but explicitly scoped out
this code/spec-table piece as "D-0018 + AC-9's job." With AC-9 gone, nothing
currently owns it.

Not a runtime correctness bug — the stale entries are documentation of a
kernel promise that has shifted, not a live behavioral gap. But it's exactly
the drift the spec-table methodology ([ADR-0011](../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md))
exists to catch, and D-0018's own "future deprecation pass" clause names
filing a fresh gap as the correct next step once a designated sweep point
falls through.

## Proposed fix

1. Remove `PreflightBranchNotFoundError` and `CodePreflightBranchNotFound`
   from `internal/verb/authorize.go` (confirm no external consumer matches
   on the type first, per D-0018's original API-stability caution).
2. Update `branch-cell-2`'s `ExpectedErrorCode` to `rung-pair-illegal` in
   `internal/workflows/spec/branch/rules.go`, and the corresponding entry in
   `internal/workflows/spec/rules.go`'s `GlobalRules()`.
3. Update the keyword-set mapping in
   `internal/policies/m0158_ac2_corner_cells_test.go` to match.
4. Re-run the spec-policy / bijection meta-tests (`internal/policies/branch_cell_bijection_test.go`
   et al.) to confirm no transitive drift.
