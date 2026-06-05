---
id: M-0158
title: Layer-4 branch-choreography spec cells + drift-policy extension
status: done
parent: E-0030
depends_on:
    - M-0102
    - M-0103
    - M-0104
    - M-0105
    - M-0106
tdd: required
acs:
    - id: AC-1
      title: internal/workflows/spec/branch/ exists with Rules and AntiRules exported
      status: met
      tdd_phase: done
    - id: AC-2
      title: Each epic corner-case (1-12) represented as named cell branch-cell-N
      status: met
      tdd_phase: done
    - id: AC-3
      title: Each override-surface row represented as a named override cell
      status: met
      tdd_phase: done
    - id: AC-4
      title: All cells satisfy schema invariants (Outcome, RejectionLayer, Sources)
      status: met
      tdd_phase: done
    - id: AC-5
      title: 'Meta-test: every cell has matching test under internal/policies/'
      status: met
      tdd_phase: done
    - id: AC-6
      title: 'Drift policy: new verb or branch finding without cell fails CI'
      status: met
      tdd_phase: done
    - id: AC-7
      title: Rules/AntiRules catalogs deterministically ordered (sorted by cell id)
      status: met
      tdd_phase: done
    - id: AC-8
      title: Package doc explains layer-4 carve-out and cites ADR-0011
      status: met
      tdd_phase: done
---

## Goal

Extend `internal/workflows/spec/` to a new `branch/` sub-package that encodes the layer-4 branch-choreography legality surface as `Rule` / `AntiRule` cells. Register every corner case enumerated in [E-0030 §"Corner cases"](epic.md) as a named cell with the relevant outcome, rejection layer, and source citation. Pair each cell with the matching positive or negative test already landed by M-0102 / M-0103 / M-0104 / M-0105 / M-0106. Extend the drift policy under `internal/policies/` to fail CI when a new branch-layer cell is added without a paired test, mirroring the M-0124 / M-0125 invariants for layers 1–3 ([ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) §"Cell-coverage commitment", §"Drift policy").

## Context

[ADR-0011](../../../docs/adr/ADR-0011-legal-workflow-spec-methodology.md) ratified the three-pass methodology and the per-cell coverage commitment for layers 1–3 (FSM, per-verb pre/post, cross-verb sequence). Its §"Scope" carves out **layer 4 — branch choreography — as E-0030's scope**, deliberately separate because the test fixture shape differs (git state, not entity state). E-0030's M-0102 through M-0106 each land one slice of the chokepoint mechanism and a paired test under `internal/policies/`. This milestone closes the loop: the tests already exist; the cells they correspond to need to be registered in the spec, the meta-test ("every cell has at least one matching test") needs to be extended to cover the new sub-package, and the drift policy needs to know about layer 4.

Without this milestone, the layer-4 tests are free-standing — they catch their respective failure modes but they're not protected against the "new branch-layer verb added without a paired cell" drift class that the methodology exists to prevent. Every future PR that touches the branch surface (a new flag on `aiwf authorize`, a refinement to the cherry-pick recognition, a new ritual that interacts with the trailer) is then a candidate for the same rot that originally motivated E-0033 / ADR-0011.

## Pre-decided design

Per E-0030 §"Design decisions" (the spec-table package layout is local to this milestone — settled here):

- **Package layout:** `internal/workflows/spec/branch/` with:
  - `rules.go` — exported `Rules() []spec.Rule` returning the layer-4 legal cells.
  - `antirules.go` — exported `AntiRules() []spec.AntiRule` returning the deliberate non-policed patterns (cherry-picks, paused-scope commits, sovereign-amended commits).
  - `spec.go` (optional) — layer-4-specific helpers if shared shape emerges across cells; absent if not.
- **Top-level integration:** `internal/workflows/spec/rules.go::Rules()` is amended to `out = append(out, branch.Rules()...)`; the same for `AntiRules()`. The existing schema-invariant drift policies (Outcome ≠ Unspecified, Illegal ⇒ RejectionLayer ≠ None, etc.) apply unchanged.
- **Cell-coverage extension:** the M-0124 meta-test (every spec cell has at least one matching test) extends to also scan tests under `internal/policies/` against `branch.Rules()` / `branch.AntiRules()`. Test-to-cell matching uses the same naming convention layers 1–3 use; this milestone documents the convention in the package doc comment if it isn't already canonical.
- **Drift policy extension:** the existing "every top-level Cobra verb is referenced by at least one rule" policy is extended with "every legality-pertinent finding code under `ClassBranchChoreography` is referenced by at least one illegal-outcome cell." The new class constant (introduced in M-0106) is the gate.
- **Cell catalog source-of-truth:** the 12 corner cases in [E-0030 §"Corner cases"](epic.md) plus the override-surface cells in §"Sovereign override surface" are the input. Each becomes one (or more, when an override variant exists) `Rule` / `AntiRule` entry. The cell ids match the corner-case numbers for traceability (`branch-cell-1` through `branch-cell-12`, plus `branch-cell-override-preflight`, `branch-cell-override-cherry-pick`, `branch-cell-override-force-amend`, `branch-cell-override-f-nnnn-waiver`).

## Out of scope

- The implementation of any of the cells the table catalogs — those land in M-0102 through M-0106. This milestone is consolidation, not new behavior.
- Layer-4 surfaces beyond what E-0030 ships (e.g., a hypothetical `aiwf cherry-pick` wrapper verb is out of scope; if and when it lands, it adds its own cell via the drift policy).
- Performance optimization of the meta-test against larger cell tables. Layer 4 adds ~16 cells; performance is not a concern at this scale.
- Migration of layers 1–3 cells into the new package layout. Layers 1–3 stay where they are.

## Dependencies

- **M-0102, M-0103, M-0104, M-0105, M-0106** — each lands the implementation and a paired test for one or more of the cells in the catalog. This milestone *registers* those cells in the spec table; the tests already exist.

## Acceptance criteria

<!-- Drafted at `aiwfx-start-milestone M-0158` time. AC seed set:

1. `internal/workflows/spec/branch/` package exists with `Rules()` and `AntiRules()` exported, registered in the top-level `spec.Rules()` / `spec.AntiRules()`.
2. Every numbered corner case from epic §"Corner cases" (1–12) is represented as a named cell. Cell id format: `branch-cell-N` matching the corner-case number.
3. Every override-surface row from epic §"Sovereign override surface" is represented as a named cell. Cell ids: `branch-cell-override-preflight`, `branch-cell-override-cherry-pick`, `branch-cell-override-force-amend`, `branch-cell-override-f-nnnn-waiver`.
4. Every cell satisfies the schema invariants (Outcome ≠ Unspecified; Illegal ⇒ RejectionLayer ≠ None; VerbTime ⇒ BlockingStrict; Legal ⇒ ExpectedErrorCode == ""; Sources cite the relevant ADR / milestone / D-NNN).
5. The meta-test (every cell has at least one matching test under `internal/policies/`) passes against the new sub-package. Cells lacking a matching test fail CI with a descriptive error citing which corner-case row or override-row is uncovered.
6. The drift policy is extended: a new top-level Cobra verb or a new `ClassBranchChoreography` finding code added without a corresponding cell fails CI. A test exercises this invariant by constructing a fixture spec with a missing rule and asserting the policy fires.
7. The `branch.Rules()` and `branch.AntiRules()` catalogs are stable across runs (deterministic ordering — sorted by cell id for human readability).
8. The package doc comment in `internal/workflows/spec/branch/spec.go` (or `rules.go`) explains the layer-4 carve-out, cites ADR-0011 §"Scope", and names the cell-id-to-test-name convention.

These ACs do not test branch behavior end-to-end — that's done by the prior milestones' tests. M-0158's tests are about the spec table's completeness and the drift-policy's vigilance.
-->

### AC-1 — internal/workflows/spec/branch/ exists with Rules and AntiRules exported

The package at [`internal/workflows/spec/branch/`](../../../internal/workflows/spec/branch/spec.go) exports `Rules() []spec.Rule` and `AntiRules() []spec.AntiRule`. Title retitled from "registered" to "exported" mid-implementation: the spec body's literal "amended to append branch.Rules() into spec.Rules()" was infeasible because the `branch` sub-package imports `spec` for the Rule/AntiRule types, so a parent-side `append` would cycle. Consumers union at the call site instead.

**Pinned by:** [`TestM0158_AC1_BranchPackageExistsWithRulesAndAntiRules`](../../../internal/policies/m0158_scaffold_test.go) — compile-time use + non-nil contract.

### AC-2 — Each epic corner-case (1-12) represented as named cell branch-cell-N

Twelve cells `branch-cell-1` through `branch-cell-12` registered in `branch.Rules()`. 1:1 mapping to E-0030 §"Corner cases" for traceability.

**Pinned by:**
- [`TestM0158_AC2_TwelveCornerCellsPresent`](../../../internal/policies/m0158_ac2_corner_cells_test.go)
- [`TestM0158_AC2_CornerCellOutcomesMatchEpic`](../../../internal/policies/m0158_ac2_corner_cells_test.go)
- [`TestM0158_AC2_IllegalCellsCarryExpectedErrorCode`](../../../internal/policies/m0158_ac2_corner_cells_test.go)

**Honest scope:** the "12 cells" framing produced 7 cells (3, 5, 6, 8, 9, 10, 11) of pure documentation — Legal corner cases with no kernel-code reference. Catalogued as [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md); refactor scheduled in M-0159.

### AC-3 — Each override-surface row represented as a named override cell

Four cells `branch-cell-override-*` registered. Pre-dispatch row deliberately has no cell (session-layer).

**Pinned by:**
- [`TestM0158_AC3_FourOverrideCellsPresent`](../../../internal/policies/m0158_ac3_override_cells_test.go)
- [`TestM0158_AC3_AllOverrideCellsAreLegalOutcome`](../../../internal/policies/m0158_ac3_override_cells_test.go)

**Honest scope:** 2 of 4 override cells (`-cherry-pick`, `-force-amend`) are byte-for-byte duplicates of `branch-cell-8` and `branch-cell-10`. Only `-preflight` and `-f-nnnn-waiver` are NEW mechanisms. Catalogued as [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md); refactor scheduled in M-0159.

### AC-4 — All cells satisfy schema invariants (Outcome, RejectionLayer, Sources)

Per-Rule invariants the M-0123 family enforces over layers 1–3 also hold over `branch.Rules()`: Outcome ≠ Unspecified, Illegal ⇒ RejectionLayer ≠ None, VerbTime ⇒ BlockingStrict, Legal ⇒ ExpectedErrorCode empty, Sources.Decision matches ADR-NNNN or D-NNNN shape.

**Pinned by:** [`TestM0158_AC4_SchemaInvariantsHoldOverBranchCells`](../../../internal/policies/m0158_ac4_schema_invariants_test.go).

### AC-5 — Meta-test: every cell has matching test under internal/policies/

Every branch cell has at least one matching test in the kernel test set (verb, check, cli/check, cli/authorize, cli/integration, policies). Match strategies: cell id literal OR per-cell curated keyword set. `branch-cell-override-f-nnnn-waiver` is the documented exception (F-NNNN milestone family outside E-0030).

**Pinned by:** [`TestM0158_AC5_EveryBranchCellHasMatchingTest`](../../../internal/policies/m0158_ac5_meta_coverage_test.go).

**Honest limitation:** substring matching means a future test with a similar token could false-positive; for duplicate cells (cell-8 vs override-cherry-pick) the same test satisfies both, weakening the "which view broke?" signal. Part of [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md).

### AC-6 — Drift policy: new verb or branch finding without cell fails CI

Every `Class: codes.ClassBranchChoreography` code declared in `internal/check/*.go` is referenced by at least one Illegal cell's `ExpectedErrorCode`. AST-scanned bidirectionally.

**Pinned by:**
- [`TestM0158_AC6_EveryClassBranchChoreographyCodeReferencedByCell`](../../../internal/policies/m0158_ac6_drift_test.go)
- [`TestM0158_AC6_DriftFiresOnFabricatedCode`](../../../internal/policies/m0158_ac6_drift_test.go) — sabotage pin: feeds fabricated code, asserts gap-detection fires.

**Honest limitation:** scanner only recognizes `codes.Code{}` struct literals; a future constructor form (`codes.NewCode(...)`) would be missed. Speculative; no constructor exists today.

### AC-7 — Rules/AntiRules catalogs deterministically ordered (sorted by cell id)

Both accessors sort by ID via `sort.SliceStable`.

**Pinned by:** [`TestM0158_AC7_RulesAndAntiRulesDeterministicallyOrdered`](../../../internal/policies/m0158_scaffold_test.go).

### AC-8 — Package doc explains layer-4 carve-out and cites ADR-0011

The package doc in [`internal/workflows/spec/branch/spec.go`](../../../internal/workflows/spec/branch/spec.go) explains the layer-4 carve-out, cites ADR-0011 §"Scope", names the cell-id convention, and documents the test-naming convention.

**Pinned by:** [`TestM0158_AC8_PackageDocCitesADR0011AndConvention`](../../../internal/policies/m0158_scaffold_test.go) — AST extraction + 4 marker substrings. Sabotage-verified.

## Work log

### Cycle 1 — AC-1 + AC-7 + AC-8 (scaffold)

Commit `1951f02a`. Added `ID string` to `spec.Rule` (kernel struct change). Created `internal/workflows/spec/branch/` with empty sorted accessors. Package doc cites ADR-0011. `TestM0123_AC1_SpecRuleStructShape` updated. Sabotage: package-doc strip (4 markers fire).

### Cycle 2 — AC-2 (12 corner-case cells)

Commit `d0c1d999`. Cells `branch-cell-1`..`branch-cell-12` with Outcome, RejectionLayer, ExpectedErrorCode for illegal cells, Sources citing ADR-0010. Sabotage: drop branch-cell-7.

### Cycle 3 — AC-3 (4 override cells)

Commit `63e90d6c`. `branch-cell-override-preflight`, `-cherry-pick`, `-force-amend`, `-f-nnnn-waiver`. All Legal. Sabotage: drop override-cherry-pick.

### Cycle 4 — AC-4 + AC-5 + AC-6 (drift policies)

Commit `fcb781fc`. Schema invariants test (AC-4); per-cell keyword meta-coverage (AC-5); ClassBranchChoreography ↔ cells bidirectional drift (AC-6) with explicit `DriftFiresOnFabricatedCode` sabotage pin.

### Post-cycle — third-pass review + honest-scope audit

Reviewer subagent verdict: APPROVE-WITH-FOLLOW-UPS, 12 findings T1-M01..T1-M12. T1-M01 addressed in-milestone (`branch-cell-override-f-nnnn-waiver.Kind` flipped from `"gap"` to `"finding"` per ADR-0003).

The Q&A on T1-M04 (override-cell duplication) triggered the **honest-scope audit** — the user pushed back on a "cross-reference comment" patch, asking what would happen if duplicate cells were removed. The audit identified that **9 of 16 cells (56%) carry no mechanical weight**:

- 5 Legal-non-override cells (3, 5, 6, 9, 11): pure documentation, no kernel-code reference.
- 2 Legal corner cells (8, 10): semantic duplicates of override cells.
- 2 override cells (-cherry-pick, -force-amend): semantic duplicates of corner cells.

Only 7 cells carry weight: 5 Illegal (1, 2, 4, 7, 12) + 2 standalone override (-preflight, -f-nnnn-waiver).

The audit also surfaced real-world failure modes the prior E-0030 milestones don't cover (shallow clones, force-pushes, branch renames, detached HEAD, missing cherry-pick gather-side, no operator UX for `aiwf-force` amend, advisory-only SKILL.md rituals).

## Decisions made during implementation

- **`ID string` field added to `spec.Rule`** (per pre-implementation Q&A). Layers 1–3 leave ID empty; only layer-4 cells populate it.
- **1:1 mapping of corner cases to cells** (per pre-implementation Q&A). Mechanical traceability between cell id and epic body. The audit later showed this over-specifies the catalog.
- **Consumer-layer aggregation instead of top-level append.** The spec body's literal append wording was infeasible due to the `branch → spec` import cycle. Documented in `branch/spec.go` package comment. AC-1 retitled mid-implementation from "registered" to "exported".
- **Honest-scope audit during wrap.** The user pushed back on "fix everything reflexively" patterns. The audit identified the over-specified catalog AND the missing real-world coverage. The user chose to ship M-0158 with the catalog AS-IS and address the substantive work in a new milestone (M-0159) rather than inflate M-0158's scope.

## Validation

- `go test -race -parallel 8 ./...` — green.
- `golangci-lint run ./...` — 0 issues.
- `aiwf check` — clean.
- Sabotage probes (4 cycles): 6 single-line regressions caught.
- Honest-scope audit catalogued findings as G-0210.

## Deferrals

The honest-scope audit produced 7 new gaps cataloguing real-world failure modes; combined with the 4 already-filed E-0030 gaps, **M-0159 consumes 11 gaps**:

- [G-0200](../../gaps/G-0200-preflight-main-only-carve-out-generalize-to-trunk-name-from-aiwf-yaml.md) — hardcoded `"main"` carve-out
- [G-0201](../../gaps/G-0201-authorize-preflight-carve-out-accepts-cross-rung-ritual-mismatches.md) — cross-rung mismatches
- [G-0202](../../gaps/G-0202-isolation-escape-cherry-pick-gather-side-implement-cli-detection.md) — cherry-pick gather-side not implemented
- [G-0203](../../gaps/G-0203-branchoracle-firstparentbranches-conflates-lookup-failed-with-no-branches.md) — oracle typed-error + fail-shut
- [G-0204](../../gaps/G-0204-branchoracle-silent-on-shallow-clones-ci-fetch-depth-1.md) — shallow-clone silent escape
- [G-0205](../../gaps/G-0205-branchoracle-silent-on-force-pushed-away-violating-commits.md) — force-push silent escape
- [G-0206](../../gaps/G-0206-branchoracle-false-positive-on-branch-renames-after-authorize.md) — branch-rename false positive
- [G-0207](../../gaps/G-0207-detached-head-handling-untested-in-preflight-and-oracle.md) — detached-HEAD untested
- [G-0208](../../gaps/G-0208-aiwf-force-amend-override-has-no-operator-ux-path.md) — `aiwf-force` amend has no UX
- [G-0209](../../gaps/G-0209-ritual-step-ordering-is-advisory-only-no-kernel-enforcement.md) — SKILL.md advisory only
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — M-0158 over-specification refactor

[M-0159](M-0159-real-world-hardening-of-branch-model-chokepoint.md) is the consumer milestone added to E-0030. The epic stays `active` until M-0159 wraps.

## Reviewer notes

**Third-pass review (subagent, opus pre-wrap):** 12 findings T1-M01..T1-M12, none blocking. T1-M01 addressed in-milestone. T1-M02 captured in this Reviewer notes section. T1-M04 triggered the honest-scope audit.

**Honest-scope audit (operator-led, post-third-pass):** the user pushed back on T1-M04's surface-level fix and asked what would happen if duplicate cells were removed. The audit identified that 9 of 16 cells carry no mechanical weight AND that the epic's existing milestones don't cover real-world failure modes (shallow clones, force-pushes, branch renames, detached HEAD, missing cherry-pick gather-side, no operator UX for `aiwf-force` amend, SKILL.md ritual-ordering advisory only).

**The operator's chosen path:** ship M-0158 with the over-specified catalog AS-IS, file gaps for everything, add M-0159 to address the gaps. M-0158 documents the over-specification honestly; M-0159 carries the substantive hardening work. E-0030 stays open until M-0159 completes.

**Lesson recorded:** AC framing matters. AC-2 and AC-3's literal wording produced a catalog inflated with documentation-as-cell. The methodology question — *"what's the actual value of a spec cell that documents a Legal pattern with no enforcement?"* — should be addressed at AC-design time, not at wrap. This wrap is the record; M-0159 corrects it.

**What this milestone ships honestly:**

- ✅ A working sub-package `internal/workflows/spec/branch/` with the layer-4 catalog structure.
- ✅ 5 mechanically-enforced illegal cells (1, 2, 4, 7, 12) whose codes the drift policy verifies.
- ✅ 2 mechanically-enforced override cells (-preflight, -f-nnnn-waiver) documenting genuinely new mechanisms.
- ✅ Drift policy catching new `ClassBranchChoreography` codes without cells.
- ✅ Schema invariants, meta-coverage, deterministic ordering, package doc.
- ⚠️ 9 cells of documentation-only or duplicate weight — [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md), refactored by M-0159.
- ❌ No coverage of real-world failure modes — G-0204..G-0209 (+ G-0200..G-0203), addressed by M-0159.

The epic's "watertight" claim applies to the synthetic-fixture test set. Against real-world git workflows, the honest answer is "M-0159 finishes the job."

