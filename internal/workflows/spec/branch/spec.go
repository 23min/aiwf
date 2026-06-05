// Package branch holds the layer-4 branch-choreography spec cells per
// [ADR-0011] §"Scope": *"layer 4 — branch choreography — as E-0030's
// scope, deliberately separate because the test fixture shape differs
// (git state, not entity state)"*. Layers 1–3 (FSM, per-verb pre/post,
// cross-verb sequence) live in the parent [spec] package; layer 4 lives
// here in its own sub-package so the closed-set is enumerable
// independently and the meta-policies can scope-match per layer.
//
// # Cell catalog
//
// The cells map 1:1 to the corner cases enumerated in E-0030's epic body
// §"Corner cases" (12 entries) plus the override-surface rows in
// §"Sovereign override surface" (4 entries). Cell ids follow two
// conventions:
//
//   - `branch-cell-N` for corner case N (1 ≤ N ≤ 12). Direct mapping to
//     the epic's numbered list — a reader looking up a cell can find the
//     prose rationale by the same number.
//   - `branch-cell-override-<mechanism>` for override-surface rows.
//     Names: `preflight`, `cherry-pick`, `force-amend`, `f-nnnn-waiver`.
//
// # Test-naming convention
//
// Per the meta-policy (M-0158/AC-5), every branch cell has at least one
// matching test under `internal/policies/`. The convention:
//
//   - For corner case N: any test whose name contains `BranchCell<N>` or
//     the cell's natural-key tokens, OR the existing milestone test for
//     the case (e.g., `TestIsolationEscape_AC1_AICommitOnMainFires` for
//     branch-cell-4).
//   - For overrides: any test naming `Override<mechanism>` or asserting
//     the relevant suppression path.
//
// The meta-test cites the convention explicitly; it does not require
// cosmetic test-renames in M-0102..M-0106. The existing tests under
// `internal/policies/` and the verb/check test packages they reference
// are the source of truth.
//
// # Top-level integration
//
// [Rules] returns `[]spec.Rule` and [AntiRules] returns `[]spec.AntiRule`.
// Consumers union the layer-4 sets with the parent `spec.Rules()` /
// `spec.AntiRules()` at the call site (drift tests, meta-tests, renderers).
// The branch sub-package imports `spec` for the `Rule` / `AntiRule` types,
// so a direct append from spec's aggregator would create an import cycle.
// The consumer-layer union is the pragmatic shape; the spec body's
// "amended to append branch.Rules()" wording is a descriptive intent
// satisfied by the union rather than a literal source edit.
//
// Both Rules and AntiRules are deterministically ordered by cell id
// (M-0158/AC-7) so renderer / diff consumers see stable output.
//
// # Drift policy
//
// A new top-level Cobra verb or a new `ClassBranchChoreography` finding
// code added without a paired cell in this package fails CI (M-0158/AC-6).
// The drift test lives in `internal/policies/` alongside the meta-test
// chain.
//
// [ADR-0011]: ../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md
package branch
