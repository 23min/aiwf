---
id: D-0022
title: M-0161/AC-9 deferred to follow-up milestone; M-0161 wraps 8/9
status: proposed
relates_to:
    - M-0161
    - G-0210
---
## Context

M-0161/AC-9 (G-0210) was scoped to refactor the layer-4 branch-choreography spec catalog under `internal/workflows/spec/branch/` from its current ~17-cell shape (M-0158 retained + 1 cell per M-0161 AC) to a 76-cell **mechanical-weight-only** catalog with bijection enforcement between cells and tests.

Specifically, the AC-9 body lines 581-666 specifies:

**Part 1 — Drop 9 documentation-only / duplicate cells from M-0158** (cells 3, 5, 6, 8, 9, 10, 11, override-cherry-pick, override-force-amend).

**Part 2 — Add 66 new cells across the M-0161 AC matrices** (4 + 17 + 7 + 7 + 7 + 9 + 7 + 9 cells covering each AC's full scenario shape, where today only 1 cell per AC is registered as the minimum drift-policy chokepoint).

**Part 3 — Build the `branchcell.Pin` registry under `//go:build testpins`** (a new test-only package surface where every E2E scenario calls `Pin(cellID, testFunctionName)` at setup to create a verifiable 1:1 mapping between cells and tests).

**Part 4 — Replace the `m0158_ac5_meta_coverage_test.go` keyword-set approach** with a bijection meta-test under `internal/policies/branch_cell_bijection_test.go` enforcing four invariants (every cell has ≥1 Pin, every Pin references an existing cell, no cell has 2+ Pins, no test pins 2+ cells).

**Part 5 — Catalog refactor verification, drift policy extension, meta-cell registration** (3 additional meta-cells for the bijection invariants themselves).

The total scope is on the order of 500-800 LOC of test infrastructure + 57 net new spec cells + replacing every existing E2E's coverage approach + deleting the existing keyword-set meta-test + adding two new policy tests + adding meta-cell registrations.

## Decision

**Defer AC-9 to its own follow-up milestone. Partially close M-0161 at 8/9 ACs met. AC-9 stays open under M-0161 until the follow-up milestone is in place; the milestone wrap explicitly carries AC-9 forward as the named residual.**

Rationale:

1. **Scope discipline.** AC-9 is structurally a separate refactor: it doesn't add new behavior — it restructures the catalog plus adds a fundamentally new test-infrastructure surface (the Pin registry + bijection meta-test). The work doesn't share a fixture surface with AC-1..AC-8 (those introduced rules; AC-9 reworks the spec table) and gains nothing from co-residency in the same milestone.

2. **Risk of incomplete delivery.** A 500-800 LOC test-infrastructure rework in the tail of a 9-AC milestone carries delivery risk that the prior 8 ACs do not. The bijection registry needs design Q&A on the build-tag shape (`testpins` tag vs `_test_helpers.go` suffix per AC-9 body line 610-611's alternative), and forces every existing E2E test file to be touched. Splitting it gives the work a fresh planning cycle and a fresh review surface.

3. **No load-bearing behavioral gap today.** The current 9-cell catalog (1 cell per AC + M-0158's retained 7) satisfies the M-0158/AC-6 ClassBranchChoreography drift invariant — every kernel ClassBranchChoreography code is referenced by at least one cell. The keyword-set meta-coverage policy at `internal/policies/m0158_ac5_meta_coverage_test.go` continues to enforce "each cell has at least one paired test". The behavioral guarantees AC-9 promises (1:1 bijection vs keyword-set's "≥1") are quality improvements, not load-bearing safety properties.

4. **Pattern precedent.** AC-5's cell-5 (D-0020) and AC-7's doctor JSON envelope (D-0021) used the same shape: a portion of the AC's spec defers to a follow-up milestone with a `D-NNN` naming the carve-out. AC-9's whole-scope deferral is the same pattern at a different cardinality.

5. **The milestone-wrap honest closure.** M-0161 ships at 8/9 with AC-9 as the named residual. The follow-up milestone (call it `M-016X`-spec-catalog-refactor) scopes the AC-9 work explicitly: 76-cell expansion, Pin registry shape decision, bijection meta-test, keyword-set retirement. The follow-up milestone references G-0210 directly as its target gap; G-0210 stays open until then.

## Concrete sequencing

- **Now (M-0161 wrap):** record this decision. AC-9 stays at `open` (NOT promoted to met) — the AC-9 deliverable was not produced in M-0161 and promoting met would be dishonest. The milestone wraps explicitly with 8/9 + AC-9 residual carve-out.
- **M-0161 promote to done:** the AC-FSM requires all ACs met to reach milestone done. Two paths to wrap honestly:
  - **(a) Cancel AC-9 with reason "deferred to follow-up per D-0022"** — clean: AC-9 transitions to terminal-cancelled; milestone done. The follow-up milestone's AC-1 scope is the AC-9 work.
  - **(b) Leave AC-9 open + wrap M-0161 to done via `--force --reason "deferral per D-0022; AC-9 carried as named residual"`** — sovereign override; honest but uses the sovereign-act surface for the FSM transition. M-0161 history records the force-promote with audit trail.
  - Either option is honest; (a) is mechanically cleaner. Choice belongs to the wrap discipline.
- **Follow-up milestone:** scope AC-9's four-part shape (M-0158 cell drop, M-0161 cell expansion, Pin registry, bijection meta-test) into its own AC matrix. Likely 3-4 ACs in the follow-up because the four parts have natural independence.

## Why not the alternatives

- **Alternative A: attempt AC-9 in this session.** Rejected — delivery risk in the tail of a long session, no design Q&A for the Pin registry build-tag shape, and forces every existing E2E touch. The "minimal viable AC-9" would be incomplete in ways the body's enforcement-claim cells would catch.
- **Alternative B: minimal AC-9 — expand to 76 cells only, defer Pin infrastructure.** Rejected — this would close the cell-count claim while leaving the bijection-discipline claim open. The catalog refactor IS the bijection discipline (per AC-9 body lines 634-640: "the bijection meta-test pins a strictly stronger claim than keyword-set"); without the Pin registry the refactor has no mechanical chokepoint distinguishing it from today's keyword-set surface. Half the AC done is worse than the AC deferred-and-tracked.
- **Alternative C: promote AC-9 met with "deferred" as the rationale.** Rejected — the AC-9 deliverable was not produced. Promoting met would dishonestly claim the catalog refactor + Pin registry exists when neither does. The AC-5 cell-5 / AC-7 doctor-JSON deferrals were partial — the bulk of those ACs DID ship. AC-9 ships nothing today; the partial-close framing doesn't apply.

## References

- M-0161/AC-9 (G-0210) — the AC this defers
- M-0161 body lines 577-694 — the AC-9 four-part scope this defers
- D-0020 (AC-5 cell-5 deferral) — partial-AC-deferral pattern precedent
- D-0021 (AC-7 doctor JSON deferral) — partial-AC-deferral pattern precedent
- [G-0210](../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap AC-9 targets; stays open until the follow-up milestone
- `internal/policies/m0158_ac5_meta_coverage_test.go` — the keyword-set meta-coverage that stays in place until AC-9's bijection replacement lands
- `internal/workflows/spec/branch/rules.go` — the catalog AC-9 was to refactor; current 1-cell-per-AC shape stays in place
