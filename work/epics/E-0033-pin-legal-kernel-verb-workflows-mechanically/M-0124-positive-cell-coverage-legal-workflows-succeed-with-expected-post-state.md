---
id: M-0124
title: 'Positive cell coverage: legal workflows succeed with expected post-state'
status: done
parent: E-0033
depends_on:
    - M-0123
    - M-0130
    - M-0131
tdd: required
acs:
    - id: AC-1
      title: 'spec.EvaluatePredicate primitive: closed Subject + Op vocabulary over Predicate'
      status: met
      tdd_phase: done
    - id: AC-2
      title: Fixture helpers + satisfyPredicate self-verification
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'Per-cell positive driver: table-driven over spec.Rules() Legal cells'
      status: met
      tdd_phase: done
    - id: AC-4
      title: 'Coverage meta-test: every Legal rule has a corresponding positive subtest'
      status: met
      tdd_phase: done
---
## Goal

For every **legal** cell in M-0123's spec table, write a test under `internal/policies/` that drives the real `aiwf` binary against a fixture tree, executes the cell's verb, and asserts:

- Exit code 0
- The post-state matches the cell's expected target (from `entity.AllowedTransitions` / `entity.CancelTarget` / `tddPhaseTransitions`)
- The commit carries the expected `aiwf-verb` / `aiwf-entity` trailers

Tests are table-driven from the spec — adding a new legal cell to the spec automatically grows the test surface.

## Test shape

Per cell:

1. Build a fixture tree that satisfies the rule's preconditions (predicates + entity states) via the in-process `cellcoverage` package (M-0137 optimization — fixture setup doesn't fork).
2. Invoke the binary via `testutil.RunBin` (subprocess — the integration seam).
3. Assert exit 0, the target's post-state, and the commit's trailers.
4. Re-load via `tree.Load` to read the post-mutation tree.

The fixture is built using kernel verbs themselves (`verb.Add`, `verb.Promote`, `verb.AddAC`, …) — not by hand-writing markdown — so the test inputs share the same legality model as the production path.

## Coverage commitment

Every legal cell in `Rules()` has at least one corresponding positive subtest. The meta-test (AC-4) walks `Rules()`, confirms each cell appears in the driver's `enumerateLegalCases` output, and confirms case names are unique. Missing coverage fails CI; case-name collisions fail CI.

## Approach

- `internal/cellcoverage` package provides `CellFixture` (isolated tmp git repo with `aiwf init` applied) plus per-kind helpers (`epicAt`, `milestoneAt`, …) that walk the FSM to a target `(Kind, FromState)`.
- The driver derives the verb invocation from the cell's `(Kind, FromState, Verb, Preconditions)` plus the FSM's allowed targets, materializing every precondition via one of three paths — verb-arg-only, verb-arg-shaped, fixture-state — so no per-cell special-casing is needed.
- Parallel-by-default; race + parallel 8 clean.

## What this milestone does *not* do

- Does not cover illegal cells (M-0125's scope).
- Does not test branch-context preconditions (E-0030's scope).
- Does not exercise random sequences (deliberately deferred).

### AC-1 — spec.EvaluatePredicate primitive: closed Subject + Op vocabulary over Predicate

### AC-2 — Fixture helpers + satisfyPredicate self-verification

### AC-3 — Per-cell positive driver: table-driven over spec.Rules() Legal cells

### AC-4 — Coverage meta-test: every Legal rule has a corresponding positive subtest

## Work log

### AC-1 — spec.EvaluatePredicate primitive
Landed the `EvaluatePredicate(p, e, t, ctx)` primitive over the closed (Subject, Op) vocabulary harvested from `Rules()` at the M-0124 ship boundary (10 atoms: `self.target-state`, `self.evidence`, `self.addressed_by`, `self.tdd_phase`, `parent.tdd`, `any-child.status`, `any-child-ac.status`, `all-children-acs.status`, plus the empty-equality / non-empty forms). Unknown Subject / Op / named-set Value return a typed error so a future rule that widens the vocabulary fails the matching atom test with a clear message. · commit 07be28e4 · 25 subtests

### AC-2 — Fixture helpers + satisfyPredicate self-verification
Landed `internal/cellcoverage` with `CellFixture`, `BringEntityToState` (per-kind FSM walks for the 7 kinds × non-terminal states), and `SatisfyPredicate` (per-atom fixture mutations + silent-drift self-verification via `EvaluatePredicate`). Fixture setup is in-process via the verb library (M-0137 optimization); subprocess fork is reserved for the cell-under-test. · commit 71d9ffdc · 27 subtests

### AC-3 — Per-cell positive driver
Landed `internal/policies/m0124_positive_driver_test.go` — the per-cell positive driver that iterates `spec.Rules()` filtered to Legal cells, derives target(s) via target-state precondition / `CancelTarget` / `AllowedTransitions` / TDD-phase FSM, and executes each cell via subprocess. 39 subtests pass (31 Legal cells × FSM multi-target expansion). Two spec drift findings closed via G-0153: ADR.superseded precondition pair (`self.superseded_by non-empty` Legal + `== ""` Illegal `adr-supersession-mutual`) and the AC.met split on `parent.tdd` so the converse of `acs-tdd-audit` is encoded as explicit preconditions on two Legal cells rather than relying on implicit overlap. · commit b6072a16 (rebased; trailer fix), closes G-0153

### AC-4 — Coverage meta-test
Landed `internal/policies/m0124_coverage_meta_test.go` — four meta-assertions over `enumerateLegalCases`:
- `TestM0124_AC4_LegalCellsAllCovered` (every Legal cell from `Rules()` appears in the enumeration)
- `TestM0124_AC4_NoExtraEnumerations` (no enumerated case lacks a Legal-cell ground)
- `TestM0124_AC4_SubtestNamesUnique` (pins the precondition-signature disambiguator)
- `TestM0124_AC4_EveryCaseHasTargets` (no case carries an empty target)

Sanity-checked by temporarily filtering out epic cells in `enumerateLegalCases` — all 4 epic Legal cells flagged with actionable error messages. · commit 20deafee

## Decisions made during implementation

- **Driver materialization model.** Every precondition is materialized via one of three paths: (a) verb-arg-only (`self.target-state`, `self.evidence`) — supplied at run-time, no fixture mutation; (b) verb-arg-shaped field (`self.addressed_by`, `self.superseded_by`) — driver appends the corresponding flag (`--by`, `--superseded-by`) and the field populates atomically with the transition; (c) fixture-state (everything else) — `SatisfyPredicate` mutates the fixture before the cell-under-test runs. The driver has no per-cell special cases.

- **`parent.tdd` materialization at fixture-build time.** Milestone TDD policy is set at `aiwf add milestone` and isn't changed afterward. The driver inspects preconditions for `parent.tdd` before fixture build and passes `BringOpts.ParentTDD` to set the field at add time, rather than mutating after the fact.

- **Multi-target expansion for promote-verb cells.** `(epic, active, promote)` reaches both `done` and `cancelled` via the FSM; the driver expands such cells into one subtest per target (faithful to `entity.AllowedTransitions`). Cancel-verb cells have single targets via `entity.CancelTarget`; AC.cancel hardcodes `cancelled` since `CancelTarget` covers top-level kinds only.

- **In-process fixture, subprocess cell-under-test.** Fixture setup uses the verb library directly (~10ms/operation); the cell-under-test runs via `testutil.RunBin` (subprocess, the integration seam) — ~80ms/operation. This M-0137 optimization keeps the 39-subtest run under 1.5s wall-time at parallel 8 (race-mode: ~6.8s).

- **Spec drift fix over driver workaround (G-0153).** Initial AC-3 carried a `prepareKernelPreconditions` function that papered over two cells the kernel rejects with flags the spec didn't encode (ADR `--superseded-by`, AC.met `tdd_phase=done` under `tdd:required`). Per user pushback ("workarounds = skimping"), the spec was corrected: ADR cell gained the precondition pair, AC.met cell was split on `parent.tdd` into two converse-of-audit Legal cells. The driver's `defeatOverlappingIllegalCells` function was removed; the unified predicate-materialization loop handles everything.

## Validation

- `go test -race -parallel 8 ./...` — all packages clean, `internal/policies` 6.8s (39 driver subtests + 4 meta-test subtests + the rest of the policies suite).
- `go test ./internal/policies/ -cover` — 78.9% coverage.
- `go test ./internal/cellcoverage/ -cover` — 75.2% coverage.
- `go test ./internal/workflows/spec/ -cover` — 68.4% coverage.
- `golangci-lint run ./internal/...` — 0 new issues (2 pre-existing `revive` warnings in `spec.go` unrelated).
- `aiwf check` — 0 errors, 27 warnings (all pre-existing).
- `aiwf show M-0124` — every AC `met` / phase `done`, no findings.

## Deferrals

None. The remaining E-0033 scope (negative cell coverage, branch-context preconditions, random-sequence exercise) is captured in dependent milestones, not deferred from M-0124.

## Reviewer notes

- **Spec corrected, not skimped.** The mid-stream G-0153 fix re-shaped two spec cells (ADR.superseded + AC.met) so the kernel's real preconditions are encoded as explicit Legal-cell preconditions rather than implicit "and the Illegal companion doesn't fire." The driver's design rule is now universal — every precondition materialized via predicate-driven setup, no per-cell special cases. `defeatOverlappingIllegalCells` was removed entirely after the fix; if a reviewer sees that function referenced anywhere, it's stale.

- **Predicate vocabulary widened by one atom.** `self.superseded_by` (symmetric to `self.addressed_by`, single-string field on the ADR entity) is now part of the closed Subject vocabulary. `spec.EvaluatePredicate` handles it via the existing `cmpString` op family; `evaluate_test.go` covers positive + negative cases for both `non-empty` and `== ""`.

- **`internal/policies/setup_test.go` env hygiene.** When the policies test suite runs under a parent git hook (e.g. pre-push, or pre-commit triggered by `aiwf promote`), git passes `GIT_DIR` / `GIT_WORK_TREE` / `GIT_COMMON_DIR` / `GIT_INDEX_FILE` / `GIT_OBJECT_DIRECTORY` down to the test binary. The cellcoverage helpers call `gitops.Init(ctx, t.TempDir())` which inherits these via `os.Environ()`, steering `git init` into the parent worktree's gitdir and producing config-lock contention against the parent `.git/config`. `TestMain` clears these vars so the test binary is insulated. The fix is hygiene for any policies test running under a parent hook; reproduced under `GIT_DIR=... go test`.

- **Multi-target expansion is FSM-faithful, not operator-restricted.** `(epic, active, promote)` reaches both `done` and `cancelled` via `entity.AllowedTransitions`. Although operators typically use `aiwf cancel` for cancellation, the FSM permits `aiwf promote E-0001 cancelled`, so the driver tests both. The duplication with `(epic, active, cancel)` is deliberate: cancel and promote use different code paths (CancelTarget vs ValidateTransition; different trailer values) and both produce valid commits.

- **Case-name disambiguator.** `caseName` appends a precondition signature (`shortAtom` representation of each precondition outside the `(Kind, FromState, Verb, target)` quadruple) when distinct Legal cells share the same key — currently only AC.met's split on `parent.tdd`. Examples: `ac-open-promote-to-met-ptddnerequired` vs `ac-open-promote-to-met-ptddeqrequired-tddphaseeqdone`. The names are functional, not pretty; if the spec grows more overlaps the signature scheme stays mechanical.

- **G-0153 closed inline.** The spec drift gap was filed on `main` then merged in via `chore/spec-drift-gap`. ID reallocation: gap-side `G-0151` → `G-0152` (main collision with `aiwf-status-worktrees` gap), then worktree-side `G-0152` → `G-0153` (archive collision with `fsm-history-consistent` gap). The fix lands within M-0124's diff (`spec/rules.go`, `spec/evaluate.go`, `spec/evaluate_test.go`).
