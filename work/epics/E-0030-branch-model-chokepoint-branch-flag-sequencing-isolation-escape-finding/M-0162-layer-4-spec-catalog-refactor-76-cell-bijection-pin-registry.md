---
id: M-0162
title: 'Layer-4 spec-catalog refactor: 76-cell bijection + Pin registry'
status: in_progress
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: 'M-0158 cell drop: remove 9 documentation-only catalog entries'
      status: met
      tdd_phase: done
    - id: AC-2
      title: branchcell.Pin registry under //go:build testpins + branchtest sub-package
      status: open
      tdd_phase: green
    - id: AC-3
      title: 'M-0161 cell expansion: organic count via bijection invariants'
      status: open
      tdd_phase: red
    - id: AC-4
      title: Bijection meta-test replaces M-0158/AC-5 keyword-set; 4 invariants
      status: open
      tdd_phase: red
---
## Goal

Land the layer-4 branch-choreography spec-catalog refactor that M-0161/AC-9 was scoped to deliver before [D-0022](../../decisions/D-0022-m-0161-ac-9-deferred-to-follow-up-milestone-m-0161-wraps-8-9.md) deferred it. The refactor brings the catalog under `internal/workflows/spec/branch/` from its current ~17-cell shape (M-0158 retained + 1 cell per M-0161/AC-1..AC-8) to a mechanical-weight-only catalog with strictly stronger 1:1 bijection meta-coverage between cells and tests, replacing the M-0158/AC-5 keyword-set approach, and introduces a test-only `branchcell.Pin` registry under build-tag isolation as the chokepoint.

E-0030 cannot honestly close until this milestone lands — the catalog discipline is part of the epic's branch-model-chokepoint deliverable scope per the epic body's §"What's settled".

## Context

[M-0161/AC-9](M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) scoped four parts of the catalog refactor. The original AC-9 sequencing (drop → expand → Pin → bijection) was revised at M-0162 reviewer pass to **infrastructure-first** sequencing (drop → Pin → cells-and-Pins-together → bijection) to fix the keyword-set policy gap during the cell-expansion window. The four parts now map to:

1. **AC-1: M-0158 cell drop** — remove 9 documentation-only / duplicate cells (branch-cells 3, 5, 6, 8, 9, 10, 11, override-cherry-pick, override-force-amend) plus their keyword-set entries.
2. **AC-2: branchcell.Pin registry** under `//go:build testpins` + dedicated `branchtest` sub-package (the Q&A-locked shape). Registry exists; no cells yet require Pin calls beyond the existing M-0158 + M-0161 chokepoint set.
3. **AC-3: cell expansion + Pin call additions in lockstep.** Each new cell ships with its Pin call from the corresponding E2E subtest in the same commit. The existing M-0159 + M-0158 + M-0161 cells already-in-catalog also gain Pin calls.
4. **AC-4: Bijection meta-test** replaces `internal/policies/m0158_ac5_meta_coverage_test.go` (keyword-set ≥1 match) with `internal/policies/branch_cell_bijection_test.go` enforcing four invariants: every cell has ≥1 Pin, every Pin references an existing cell, no cell has 2+ Pins, no test pins 2+ cells.

The current state stays load-bearing through AC-3: the existing 1-cell-per-AC catalog satisfies M-0158/AC-6's `ClassBranchChoreography` drift invariant, and `m0158_ac5_meta_coverage_test.go` continues to enforce the keyword-set ≥1 paired-test claim until AC-4 deletes it. No load-bearing safety property is missing — the refactor is a quality / discipline upgrade.

## Scope

This milestone implements all four parts of D-0022's deferred scope. The AC matrix below partitions the work for natural sequencing; each AC is independently testable + verifiable. Total estimated delivery: ~500-800 LOC test infrastructure + ~57 net new spec cells + ~30 E2E test files touched for Pin call additions + 2 policy file changes (delete + add) + 3 meta-cell registrations.

## Dependencies

- M-0161 (done) — the eight ACs whose matrices this milestone expands to cell form.
- M-0158 (done) — the catalog whose 9 doc-only cells this milestone drops.

## Out of scope

- **Authorize-side ordering enforcement** (the G-0209 residual): the AC-8 carve-out for the implicit-current authorize path stays open as operator-discipline. A future kernel decision may extend the rule.
- **Per-AC behavioral changes**: this milestone restructures the catalog and tightens the meta-coverage; the underlying rules (AC-1..AC-8) and their pass/fail behavior are unchanged.
- **`branchcell.Pin` build-tag shape decision** was settled at M-0162 Q&A as `//go:build testpins` + dedicated `branchtest` sub-package. The M-0161/AC-9 body's `_test_helpers.go` alternative was found incorrect (the suffix does not actually exclude files from production builds).

## Acceptance criteria

### AC-1 — M-0158 cell drop: remove 9 documentation-only catalog entries

**Observable behavior.** The layer-4 branch-choreography catalog at `internal/workflows/spec/branch/rules.go` no longer contains 9 documentation-only / duplicate cells per [M-0161/AC-9 body §"Part 1"](M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) (lines 581-590). The keyword-set entries for the dropped cells at `internal/policies/m0158_ac5_meta_coverage_test.go` are removed in the same commit so the still-active meta-coverage policy stays green. The remaining catalog continues to satisfy M-0158/AC-6's `ClassBranchChoreography` drift invariant.

**Cells dropped (9):**

- 5 legal-non-override documentation-only cells: `branch-cell-3`, `branch-cell-5`, `branch-cell-6`, `branch-cell-9`, `branch-cell-11`
- 2 legal-AND-override cells (semantic duplicates of override cells): `branch-cell-8`, `branch-cell-10`
- 2 override-named cells (semantic duplicates of corner-case cells): `branch-cell-override-cherry-pick`, `branch-cell-override-force-amend`

**Cells retained from M-0158 (7):**

- 5 illegal-outcome cells with real mechanical weight: `branch-cell-1`, `branch-cell-2`, `branch-cell-4`, `branch-cell-7`, `branch-cell-12`
- 2 standalone override cells: `branch-cell-override-preflight`, `branch-cell-override-f-nnnn-waiver`

**Mechanical assertions:**

1. **Drop-set verification.** A test under `internal/policies/m0162_ac1_drop_test.go` asserts each of the 9 cell IDs above is ABSENT from `branch.Rules()`. Each absence is a separate subtest so a regression that re-adds one of the 9 fires loudly at the offending cell.

2. **Retained-set verification.** The same test asserts each of the 7 retained M-0158 cell IDs is PRESENT in `branch.Rules()`. Catches a future change that drops one of the load-bearing cells alongside cleanup.

3. **Keyword-set meta-coverage stays green.** The 9 dropped cells' entries in `internal/policies/m0158_ac5_meta_coverage_test.go::keywords` (lines 60-77 today) are removed in the same AC-1 commit so the keyword-set policy continues to pass with the new catalog. AC-4 deletes the keyword-set file entirely.

4. **Sabotage-verifiable.** Re-adding a dropped cell to `branch.Rules()` fires the absence subtest; removing a retained cell fires the presence subtest; removing the keyword-set entry update fires the keyword-set policy on the dropped cell IDs. The discriminating tests fire either way.

**Scope of closure (honest).** AC-1 partial-closes G-0210 only — the 9-cell drop is one of the four parts G-0210 names. G-0210 stays open until AC-2 (Pin registry), AC-3 (cell expansion + Pin lockstep), and AC-4 (bijection meta-test) all land.

**Edge cases:**

- **M-0161-era cells stay registered.** AC-1 drops only M-0158-era doc-only cells; the 1-cell-per-AC chokepoints M-0161 added (`branch-cell-isolation-escape-oracle-failure`, `-shallow-clone`, `-orphaned-ai-commit`, `-rename-survival`, `-id-rename-untrailered`, `-detached-head-preflight`, `-promote-on-wrong-branch`) all stay — they're part of the M-0161/AC-6 drift-policy chokepoint surface.
- **Meta-coverage transition.** Between AC-1 and AC-4, `m0158_ac5_meta_coverage_test.go` is the active meta-coverage; the keyword-set entries for the 9 dropped cells need removal in the same AC-1 commit so the meta-test stays green. AC-4 deletes the file entirely.

**References.**

- M-0161/AC-9 body §"Part 1" — the inherited drop list this AC discharges
- [M-0158](M-0158-layer-4-branch-choreography-spec-cells-drift-policy-extension.md) — the catalog whose doc-only cells this AC drops
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap this AC partial-addresses (closes G-0210 once AC-2..AC-4 land)
- `internal/workflows/spec/branch/rules.go` — the catalog the AC touches
- `internal/policies/m0158_ac5_meta_coverage_test.go` — keyword-set meta-coverage that stays in place through AC-3, removed in AC-4

### AC-2 — branchcell.Pin registry under //go:build testpins + branchtest sub-package

**Observable behavior.** A new test-only package `internal/workflows/spec/branch/branchtest` introduces a `Pin(cellID, testFunctionName string)` registry callable from any test under the `//go:build testpins` build tag. The registry accumulates pins for later inspection by AC-4's bijection meta-test. The package + its single source file `pin.go` carry the `//go:build testpins` header so production `go build` omits both.

CI runs and the Makefile's `test-pins` target carry `-tags testpins`; bare `go test ./...` without the tag silently skips the pin-calling tests and the bijection meta-test (the latter also tagged). The build-tag convention is documented in the package doc comment of `pin.go` itself — kept next to the symbol to minimize drift (per reviewer T-fix; no separate README).

**Per the M-0162 Q&A decision §"Pin shape" (locked at AC-body-authoring time):** option 1 (`//go:build testpins + dedicated branchtest sub-package`) was selected over the AC-9 body's `_test_helpers.go` alternative (which was found incorrect — that suffix doesn't actually keep files out of production). The branchtest sub-package gives the test-only nature an import-path-level marker AND the build tag enforces link-time exclusion.

**API shape:**

```go
//go:build testpins

// Package branchtest provides the Pin registry used by AC-3's
// cell-expansion E2E tests and AC-4's bijection meta-test. The
// package and its symbols are compiled only when -tags testpins
// is set; production `go build` omits them entirely.
//
// Usage:
//   func TestX_AC3_Foo(t *testing.T) {
//       branchtest.Pin("branch-cell-foo", t.Name())
//       ...
//   }
//
// The bijection meta-test at internal/policies/branch_cell_bijection_test.go
// inspects the registry after every E2E test in the test-pins
// build completes.
package branchtest

import "sync"

// Pin records that a test function exercises a specific
// branch.Rules() cell. Calls accumulate into a process-local
// registry inspected by the bijection meta-test at AC-4.
//
// Calls from tests inside `t.Run` should pass t.Name() so the
// subtest's full name (TestX/sub-row) appears in the registry.
func Pin(cellID, testName string) { ... }

// Pins returns a snapshot of accumulated pins. Used by the
// bijection meta-test at internal/policies/.
func Pins() map[string][]string { ... }
```

**Mechanical assertions:**

1. **Build-tag exclusion.** A test under `internal/policies/m0162_ac2_build_tag_test.go` builds the production `aiwf` binary without `-tags testpins` and asserts the `branchtest` package symbols are NOT present in the resulting binary. Concrete pattern: `go build -o /tmp/aiwf-no-pins ./cmd/aiwf && go tool nm /tmp/aiwf-no-pins | grep -c '/branch/branchtest/' == 0`. Sabotage-verified by removing the build-tag header on `pin.go`.

2. **API existence verification.** A test under the testpins tag asserts `Pin()` accepts the two-string signature and `Pins()` returns the accumulated `map[string][]string` shape. Catches a future refactor that changes the API.

3. **Package-doc presence.** A structural test asserts `pin.go`'s package doc comment contains the strings `//go:build testpins` and `branchtest.Pin(` (the usage code-fence). Ensures the build-tag convention stays AI-discoverable per CLAUDE.md "Kernel functionality must be AI-discoverable." The doc lives next to the symbol; no separate README needed.

4. **Sabotage-verifiable.** Removing the build-tag header fires the build-tag exclusion test (symbols appear in production); removing the API surface fires the existence test; removing the package doc fires the doc-presence test.

**Note.** AC-2's deliverable is the registry infrastructure only. The Pin call sites in E2E tests are AC-3's deliverable (cell expansion + Pin calls land together — the infrastructure-first sequencing fix per the reviewer-locked B1 resolution).

**Edge cases:**

- **Pin call inside parallel subtests.** Subtests calling `t.Parallel()` run concurrently; the Pin registry uses a `sync.Mutex` around the accumulator to be data-race-free. The test-only nature means a sync.Mutex import in test code is acceptable.
- **Pin call from inside `Setup` vs `t.Run` body.** Both shapes supported by passing `t.Name()` explicitly. The M-0159 RunScenarios framework calls `Setup` after entering the subtest's `t.Run` so `t.Name()` resolves to the subtest's full path.
- **Newcomer running bare `go test`.** Without the `testpins` tag, the registry is empty and the bijection meta-test (also tagged) is skipped. CI and the Makefile carry the tag; local newcomers see no pin-related output. The CI workflow change to carry `-tags testpins` ships in AC-4 alongside the bijection meta-test (since the registry alone is harmless without the test that consumes it).

**References.**

- M-0161/AC-9 body §"Part 3" — the inherited Pin registry scope this AC discharges
- M-0162 Q&A §"Pin shape" — the build-tag + branchtest sub-package decision
- M-0162 reviewer pass §B1 — the AC-2/AC-3 swap rationale (Pin registry before cell expansion so Pins and cells ship together at AC-3)
- AC-3 (this milestone) — the cell expansion + Pin call lockstep that consumes the registry
- AC-4 (this milestone) — the bijection meta-test that consumes the Pin registry

### AC-3 — M-0161 cell expansion: organic count via bijection invariants

**Observable behavior.** The branch-choreography catalog at `internal/workflows/spec/branch/rules.go` is expanded with one cell per discriminating E2E subtest across the full test surface (M-0158 retained + M-0159 era + M-0161 ACs 1-8). Each new cell ships **with its `branchtest.Pin(cellID, t.Name())` call from the corresponding subtest in the same commit** — the infrastructure-first sequencing locked at M-0162 reviewer pass §B1.

The exact cell count is determined organically by subtest discrimination — the deliverable is bijection-invariant readiness (every E2E subtest pins exactly one cell; every new cell carries exactly one Pin), not arithmetic matching to the M-0161/AC-9 body's "66 new cells" forecast. The actual count is reported at AC-4 wrap.

**Cells touched (organic count; ~57-77 expected):**

The M-0161 AC bodies define the matrix shapes (these counts are the **expected upper bound** based on per-subtest discrimination; the actual deliverable is bijection readiness, not arithmetic):

- M-0161/AC-1 — ~4 trunk-name shapes (TestAuthorize_AC1_NonMainTrunkNames_Accept subtests)
- M-0161/AC-2 — ~16 rung-pair cells + 1 override (TestAuthorize_AC2_RungPair_Matrix subtests)
- M-0161/AC-3 — ~13 oracle-state subtests + 2 sovereign-override subtests
- M-0161/AC-4 — ~11 shallow-clone subtests + 2 sovereign-override subtests
- M-0161/AC-5 — ~7 force-push-orphan subtests + 1 cell-7 reflog-disabled composition subtest
- M-0161/AC-6 — ~9 rename-resolution subtests
- M-0161/AC-7 — ~7 detached-HEAD subtests (B1 follow-up included)
- M-0161/AC-8 — ~8 promote-on-wrong-branch subtests

**Plus Pin calls added to existing M-0158 + M-0159-era E2E subtests so they reference the cells already in `branch.Rules()`** (reviewer §B4 clarification — the bijection invariants apply across the full catalog, not just M-0161 cells):

- `branch_scenarios_ac4_test.go` (M-0159/AC-4 ack scenarios) → Pin to `branch-cell-id-rename-untrailered` and AC-4-era illegal cells already in `branch.Rules()`.
- `branch_scenarios_ac5_test.go` (M-0159/AC-5 trailer-verb-unknown) → Pin to the M-0159 trailer-verb cells.
- `branch_scenarios_ac6_test.go` (M-0159/AC-6 cherry-pick) → Pin to `branch-cell-8` / `branch-cell-override-cherry-pick` (the latter is in AC-1's drop list — after AC-1, only `branch-cell-8` remains for the cherry-pick semantics; but `branch-cell-8` is ALSO in AC-1's drop list... resolved in cycle: AC-3 may need to retain one of cells 8/10 if M-0159-era tests depend on them).

**Per CLAUDE.md "Don't paper over a test failure":** if AC-3's Pin-wiring round reveals that M-0158-era tests genuinely need the dropped cells, AC-3 either (a) reverses the AC-1 drop for that cell with explicit justification, or (b) retitles the M-0159 test to reference a retained cell with equivalent semantics. The cycle's discriminating signal is "what does the test actually exercise" — pin to the cell whose mechanical claim matches the test's assertions.

Each new or M-0161-era subtest gets exactly one Pin call to exactly one cell. AC-2 provides the call surface; AC-3 wires the cell entries + Pin calls together; AC-4 enforces the bijection across the full catalog.

**Mechanical assertions:**

1. **Cell-presence verification.** A test under `internal/policies/m0162_ac3_expanded_set_test.go` asserts each E2E test function's expected cell IDs are present in `branch.Rules()`. The test parses the E2E files for Pin call sites and matches them against `branch.Rules()` entries.

2. **Pin-call structural presence.** A grep-style assertion at `internal/policies/m0162_ac3_pin_presence_test.go` (renamed from AC-2 per the swap) walks every file under `internal/cli/integration/` matching the AC-3 surface list and asserts each file has at least one `branchtest.Pin(...)` call. This is **structural coverage only**; behavioral discrimination (that the Pin call actually accumulates into the registry) ships at AC-4's bijection invariants.

3. **Subtest-to-cell mapping.** Every E2E subtest under `internal/cli/integration/branch_scenarios_*.go`, `isolation_escape_*.go`, `detached_head_*.go`, and `promote_wrong_branch_*.go` calls `branchtest.Pin(cellID, t.Name())` at setup. AC-3's cell-set must cover every Pin call site.

4. **Keyword-set entries added in lockstep.** AC-3 adds keyword-set entries to `m0158_ac5_meta_coverage_test.go` for every new cell — required so the still-active meta-coverage policy stays green through AC-3. This is the throwaway work per reviewer §B1: the entries get deleted at AC-4 alongside the file. **Cost ~57 entries; cheap edits to one file; required for sequencing coherence.**

5. **Sabotage-verifiable.** Removing a cell that an E2E subtest references makes the cell-presence test fail naming the orphan cell; removing the Pin call from a subtest fires AC-4's bijection invariant #1 (post-AC-4); adding a cell without a Pin call fires the cell-presence assertion.

**Scope of closure (honest).**

- **M-0161/AC-5 cell-5 deferred** per [D-0020](../../decisions/D-0020-m-0161-ac-5-cell-5-orphan-acknowledgment-deferred-to-verb-extension.md): the orphan-acknowledgment composition is unshippable until `aiwf acknowledge-illegal` extends to handle unreachable SHAs (tracked at G-0226). AC-3 does NOT add a cell for cell-5; the gap is preserved. AC-4's bijection invariants tolerate this because neither the cell nor the Pin exists.
- **M-0161/AC-8 cell-6 (detached HEAD on promote) deferred** per the AC-8 body's in-test carve-out (no D-NNN; documented in `promote_wrong_branch_scenarios_test.go`). AC-3 may file a new D-NNN to elevate this carve-out to the same status as D-0020 (one of: file new D-0023 / leave as in-test comment / consolidate at AC-4 wrap). Default: file the D-NNN at AC-3 cycle Q&A for symmetry with D-0020.

**Catalog count reported, not pinned.** The AC-4 wrap report records the actual cell count. The M-0161/AC-9 body's "76 total" forecast is a planning estimate, not a contract. If the actual count is 73 or 80, the discharge is honest.

**Edge cases:**

- **AC-2 prerequisite.** AC-3 depends on the Pin registry being available; per the swap, AC-2 lands first.
- **M-0159 framework subtests.** Some matrix-level tests use `RunScenarios([]Scenario{...})` producing per-row subtests via `t.Run`. The Pin call goes inside the Scenario's Setup function so each subtest pins its own cell.
- **Tests that exercise multiple cells.** Per AC-4 invariant #4 (no test pins 2+ cells), a single test function exercising distinct cells must split into subtests, each pinning its own cell. Reviewer T2 noted this may force migrations during AC-3 — those migrations are part of the AC-3 deliverable.

**References.**

- M-0161/AC-9 body §"Part 2" — the inherited expansion scope this AC discharges
- M-0162 reviewer pass §B1 — the AC-2/AC-3 swap rationale
- M-0162 reviewer pass §B4 — the M-0159-era cell inclusion clarification
- M-0161 AC bodies — the matrix shapes that determine the per-AC cell counts
- AC-2 (this milestone) — the Pin registry prerequisite
- AC-4 (this milestone) — the bijection meta-test that validates AC-3's expansion correctness
- [D-0020](../../decisions/D-0020-m-0161-ac-5-cell-5-orphan-acknowledgment-deferred-to-verb-extension.md) — AC-5 cell-5 deferral preserved
- `internal/cli/integration/branch_scenarios_*.go` + sibling files — the E2E surface AC-3 references

### AC-4 — Bijection meta-test replaces M-0158/AC-5 keyword-set; 4 invariants

**Observable behavior.** A new bijection meta-test at `internal/policies/branch_cell_bijection_test.go` (under `//go:build testpins`) enforces four invariants between `branch.Rules()` and the `branchtest.Pins()` registry. The existing keyword-set meta-coverage at `internal/policies/m0158_ac5_meta_coverage_test.go` is removed in the same commit; the bijection meta-test pins a strictly stronger claim than the keyword-set's ≥1 match per AC-9 body lines 634-640.

Three meta-cells are registered for the bijection invariants themselves so the catalog records its own enforcement chokepoints alongside the rule chokepoints. The CI workflow file at `.github/workflows/go.yml` is updated to carry `-tags testpins` on the test step so the bijection meta-test runs in CI; the existing race-mode `-parallel 8` cap from `internal/policies/race_parallel_cap.go` composes cleanly (the bijection test reads a sync.Mutex-guarded registry post-test, no parallelism interaction).

**Invariants enforced (each as a separate subtest, each sabotage-verifiable):**

1. **Every cell in `branch.Rules()` has at least one Pin.** Sabotage: remove a Pin from a test → cell-with-no-Pin subtest fires naming the cell.
2. **Every Pin references a cell that exists in `branch.Rules()`.** Sabotage: add a Pin for a non-existent cell → orphan-Pin subtest fires naming the cellID.
3. **No cell has 2+ Pins.** Sabotage: add a 2nd Pin to an existing cell → double-mapping subtest fires naming both Pin call sites.
4. **No test function pins 2+ cells.** Sabotage: add a 2nd Pin from a test function → overload subtest fires naming the test and the cells.

**Meta-cells registered (3):**

- `branch-cell-meta-bijection-enforced` — positive cell documenting that the 1:1 bijection holds across all cells.
- `branch-cell-meta-pin-orphan-detected` — positive cell documenting that orphan Pin detection produces a finding.
- `branch-cell-meta-cell-orphan-detected` — positive cell documenting that cell-with-no-Pin detection produces a finding.

The meta-cells satisfy AC-4's own bijection requirement: each has a Pin from the corresponding sabotage subtest.

**M-0158/AC-5 retirement:**

- `internal/policies/m0158_ac5_meta_coverage_test.go` is **deleted** in the same commit as the bijection meta-test lands. The keyword-set ≥1-match invariant is subsumed by invariant #1 (every cell has ≥1 Pin, with the tightening to exactly 1).
- M-0158/AC-5's promoted-met status remains valid because the bijection meta-test maintains and strictly strengthens every guarantee the keyword-set asserted.
- A structural test asserts `internal/policies/m0158_ac5_meta_coverage_test.go` does NOT exist (prevents reintroduction).
- Reviewer T2 noted that the strictly-stronger invariant tightening (from ≥1 to exactly 1) may have already forced migrations during AC-3 (a test legitimately covering two cells per the keyword-set forced to split into subtests at AC-3 cycle). The AC-3 wrap report names any such migrations; AC-4 inherits a clean bijection-ready state.

**Drift policy extension:**

- The existing M-0158 drift policy at `internal/policies/m0158_ac6_drift_test.go` continues to assert "every ClassBranchChoreography code referenced by an Illegal cell"; AC-4 leaves it alone.
- A new policy at `internal/policies/m0162_ac4_drift_test.go` asserts the bijection holds at CI time (consumed by every CI run via the `testpins` tag). Adding a cell to `branch.Rules()` without a Pin OR adding a Pin without a matching cell fails CI.

**Mechanical assertions:**

1. **Four-invariant bijection meta-test.** The four subtests above. Each fails on its specific sabotage; CI runs them under `-tags testpins`.

2. **Keyword-set deletion verification.** A structural test asserts `internal/policies/m0158_ac5_meta_coverage_test.go` does NOT exist. Catches a future change that re-adds the file (e.g., a confused merge).

3. **Meta-cell registration verification.** A test asserts the 3 meta-cells exist in `branch.Rules()` and each has at least one Pin (closing the meta-coverage loop: the bijection enforcer is itself a Pinned cell).

4. **Sabotage discrimination.** Each of the 4 invariants has a paired sabotage test that constructs a fixture violating the invariant and asserts the production invariant test fires. The sabotage tests are themselves tagged `testpins`.

5. **CI tag verification.** The CI workflow (`.github/workflows/go.yml`) is updated to add `-tags testpins` to the test step. Without it, the bijection meta-test silently skips; with it, the bijection invariants are enforced. The existing race-mode `-parallel 8` cap stays in place; the bijection test is post-parallel-tests (reads the accumulator after all tests complete via a sentinel ordering) so the cap composes cleanly.

**Edge cases:**

- **AC-2 + AC-3 prerequisites.** AC-4 cannot land without AC-2's registry being available + AC-3's Pin calls being wired. Per the locked AC ordering, both land first.
- **AC-3 prerequisite (cells expanded).** AC-4 enforces the bijection over AC-3's expanded catalog. If AC-3 ships an under-expanded catalog (some E2E subtests without paired cells), AC-4's invariant #2 (orphan-Pin detection) fires at CI time and AC-3 returns for additional cells. This is the discipline working as designed.
- **M-0161/AC-5 cell-5 + M-0161/AC-8 cell-6 deferrals.** Both have no test function (the deferred cells genuinely don't exist). AC-4's invariants tolerate the gap because neither the cell nor the Pin exists — invariant #1 doesn't fire (no cell to find unpinned); invariant #2 doesn't fire (no orphan Pin); invariants #3 and #4 don't apply. The deferrals stay deferrals; AC-4 doesn't force the deferred scope.
- **Test-time pin accumulation race.** The Pin registry's mutex (AC-2) ensures the accumulator is data-race-free under `t.Parallel`. AC-4's meta-test reads the accumulator after all tests in the package have completed (via TestMain ordering or a final-stage test); reading is safe because no Pin calls are in flight.

**References.**

- M-0161/AC-9 body §"Part 4" + §"Part 5" — the inherited bijection scope this AC discharges
- AC-2 + AC-3 (this milestone) — the Pin registry + Pinned cell catalog AC-4 enforces invariants over
- `internal/policies/m0158_ac5_meta_coverage_test.go` — the keyword-set file AC-4 deletes
- `internal/policies/m0158_ac6_drift_test.go` — the existing drift policy AC-4 leaves intact
- `.github/workflows/go.yml` — the CI workflow file AC-4 updates with `-tags testpins`
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap this AC closes (full closure when AC-1, AC-2, AC-3, AC-4 all land)

## References

- M-0161 (parent epic E-0030) §"AC-9" body lines 577-694 — the inherited spec this milestone delivers.
- [D-0022](../../decisions/D-0022-m-0161-ac-9-deferred-to-follow-up-milestone-m-0161-wraps-8-9.md) — the deferral decision this milestone discharges.
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap this milestone closes.
- [M-0158](M-0158-layer-4-branch-choreography-spec-cells-drift-policy-extension.md) — the catalog whose cells this milestone drops + expands.
- `internal/workflows/spec/branch/rules.go` — the catalog file the refactor touches.
- `internal/policies/m0158_ac5_meta_coverage_test.go` — the keyword-set meta-test this milestone removes.
- M-0162 reviewer pass (subagent, 2026-06-04) — the AC-body review that fed the B1-B4 + T1-T4 fixes.
