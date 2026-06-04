---
id: M-0162
title: 'Layer-4 spec-catalog refactor: 76-cell bijection + Pin registry'
status: draft
parent: E-0030
tdd: required
---
## Goal

Land the layer-4 branch-choreography spec-catalog refactor that M-0161/AC-9 was scoped to deliver before [D-0022](../../decisions/D-0022-m-0161-ac-9-deferred-to-follow-up-milestone-m-0161-wraps-8-9.md) deferred it. The refactor brings the catalog under `internal/workflows/spec/branch/` from its current ~17-cell shape (M-0158 retained + 1 cell per M-0161/AC-1..AC-8) to a 76-cell mechanical-weight-only catalog, replaces the M-0158/AC-5 keyword-set meta-coverage with a strictly stronger 1:1 bijection between cells and tests, and introduces a test-only `branchcell.Pin` registry under build-tag isolation as the chokepoint.

E-0030 cannot honestly close until this milestone lands — the catalog discipline is part of the epic's branch-model-chokepoint deliverable scope per the epic body's §"What's settled".

## Context

[M-0161/AC-9](M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) scoped four parts of the catalog refactor:

1. **M-0158 cell drop** — remove 9 documentation-only / duplicate cells (branch-cells 3, 5, 6, 8, 9, 10, 11, override-cherry-pick, override-force-amend).
2. **M-0161 cell expansion** — add 66 cells representing the full matrix shape of each of M-0161's 8 ACs (4 + 17 + 7 + 7 + 7 + 9 + 7 + 9 = 66), where M-0161 itself registered only 1 cell per AC as the M-0158/AC-6 drift-policy minimum.
3. **Pin registry** — introduce `internal/workflows/spec/branch/pin.go` (or `pin_test_helpers.go`) under `//go:build testpins` (or `_test.go` suffix per the AC-9 body line 610's alternative shape Q&A) where every E2E scenario calls `Pin(cellID, testFunctionName)` at setup. The registry is test-only by Go convention — never compiled into production binaries.
4. **Bijection meta-test** — replace `internal/policies/m0158_ac5_meta_coverage_test.go` (keyword-set ≥1 match) with `internal/policies/branch_cell_bijection_test.go` enforcing four invariants: every cell has ≥1 Pin, every Pin references an existing cell, no cell has 2+ Pins, no test pins 2+ cells.

The current state stays load-bearing in the meantime: the existing 1-cell-per-AC catalog satisfies M-0158/AC-6's `ClassBranchChoreography` drift invariant, and `m0158_ac5_meta_coverage_test.go` continues to enforce the keyword-set ≥1 paired-test claim. No load-bearing safety property is missing — the refactor is a quality / discipline upgrade.

## Scope

This milestone implements all four parts of D-0022's deferred scope. The AC matrix below partitions the work for natural sequencing; each AC is independently testable + verifiable. Total estimated delivery: ~500-800 LOC test infrastructure + 57 net new spec cells + ~30 E2E test files touched for Pin call additions + 2 policy file changes (delete + add) + 3 meta-cell registrations.

## Dependencies

- M-0161 (done) — the eight ACs whose matrices this milestone expands to cell form.
- M-0158 (done) — the catalog whose 9 doc-only cells this milestone drops.

## Out of scope

- **Authorize-side ordering enforcement** (the G-0209 residual): the AC-8 carve-out for the implicit-current authorize path stays open as operator-discipline. A future kernel decision may extend the rule.
- **Per-AC behavioral changes**: this milestone restructures the catalog and tightens the meta-coverage; the underlying rules (AC-1..AC-8) and their pass/fail behavior are unchanged.
- **`branchcell.Pin` build-tag shape decision**: design Q&A at AC-3 of this milestone picks between `//go:build testpins` and `_test_helpers.go` suffix; both options keep the registry out of production binaries cleanly.

## Acceptance criteria

(To be authored contract-first before promote to `in_progress`, per the M-0160-established discipline. Provisional AC outline:)

- **AC-1** — M-0158 cell drop: 9 doc-only cells removed; catalog count drops by 9; meta-coverage approach continues to assert each remaining cell has a paired test.
- **AC-2** — M-0161 cell expansion: 57 new cells added (66 total per-AC matrix cells, minus the 9 already registered as 1-per-AC chokepoint cells). Catalog count reaches 73 (7 retained M-0158 + 66 M-0161 expanded).
- **AC-3** — `branchcell.Pin` registry: design Q&A picks the build-tag shape; the registry lives under that shape; runtime no-op outside test-tag builds; ~30 E2E tests call `Pin(cellID, "TestX")` at setup.
- **AC-4** — Bijection meta-test: 4 invariants enforced via subtests; each subtest sabotage-verified by inserting fixture violations; `m0158_ac5_meta_coverage_test.go` removed; meta-cells (3) registered for the bijection invariants themselves. Catalog count reaches 76.

The contract-first authoring discipline applies: AC bodies get filled before this milestone promotes to `in_progress`, sabotage-verifiable assertions explicitly named, deferral D-NNNs for any sub-scope that can't ship in this cycle.

## References

- M-0161 (parent epic E-0030) §"AC-9" body lines 577-694 — the inherited spec this milestone delivers.
- [D-0022](../../decisions/D-0022-m-0161-ac-9-deferred-to-follow-up-milestone-m-0161-wraps-8-9.md) — the deferral decision this milestone discharges.
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap this milestone closes.
- [M-0158](M-0158-layer-4-branch-choreography-spec-cells-drift-policy-extension.md) — the catalog whose cells this milestone drops + expands.
- `internal/workflows/spec/branch/rules.go` — the catalog file the refactor touches.
- `internal/policies/m0158_ac5_meta_coverage_test.go` — the keyword-set meta-test this milestone removes.
