---
id: M-0158
title: Layer-4 branch-choreography spec cells + drift-policy extension
status: in_progress
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
      title: internal/workflows/spec/branch/ exists with Rules and AntiRules registered
      status: met
      tdd_phase: done
    - id: AC-2
      title: Each epic corner-case (1-12) represented as named cell branch-cell-N
      status: open
      tdd_phase: red
    - id: AC-3
      title: Each override-surface row represented as a named override cell
      status: open
      tdd_phase: red
    - id: AC-4
      title: All cells satisfy schema invariants (Outcome, RejectionLayer, Sources)
      status: open
      tdd_phase: red
    - id: AC-5
      title: 'Meta-test: every cell has matching test under internal/policies/'
      status: open
      tdd_phase: red
    - id: AC-6
      title: 'Drift policy: new verb or branch finding without cell fails CI'
      status: open
      tdd_phase: red
    - id: AC-7
      title: Rules/AntiRules catalogs deterministically ordered (sorted by cell id)
      status: met
      tdd_phase: done
    - id: AC-8
      title: Package doc explains layer-4 carve-out and cites ADR-0011
      status: open
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

### AC-1 — internal/workflows/spec/branch/ exists with Rules and AntiRules registered

### AC-2 — Each epic corner-case (1-12) represented as named cell branch-cell-N

### AC-3 — Each override-surface row represented as a named override cell

### AC-4 — All cells satisfy schema invariants (Outcome, RejectionLayer, Sources)

### AC-5 — Meta-test: every cell has matching test under internal/policies/

### AC-6 — Drift policy: new verb or branch finding without cell fails CI

### AC-7 — Rules/AntiRules catalogs deterministically ordered (sorted by cell id)

### AC-8 — Package doc explains layer-4 carve-out and cites ADR-0011

