---
id: M-0162
title: 'Layer-4 spec-catalog refactor: 76-cell bijection + Pin registry'
status: draft
parent: E-0030
tdd: required
acs:
    - id: AC-1
      title: 'M-0158 cell drop: remove 9 documentation-only catalog entries'
      status: open
      tdd_phase: red
    - id: AC-2
      title: 'M-0161 cell expansion: organic count via bijection invariants'
      status: open
      tdd_phase: red
    - id: AC-3
      title: branchcell.Pin registry under //go:build testpins + branchtest sub-package
      status: open
      tdd_phase: red
    - id: AC-4
      title: Bijection meta-test replaces M-0158/AC-5 keyword-set; 4 invariants
      status: open
      tdd_phase: red
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

### AC-1 — M-0158 cell drop: remove 9 documentation-only catalog entries

**Observable behavior.** The layer-4 branch-choreography catalog at `internal/workflows/spec/branch/rules.go` no longer contains 9 documentation-only / duplicate cells per [M-0161/AC-9 body §"Part 1"](../M-0161-imagination-driven-hardening-shallow-force-push-rename-detached-trunk.md) (lines 581-590). The remaining catalog continues to satisfy M-0158/AC-6's `ClassBranchChoreography` drift invariant and `m0158_ac5_meta_coverage_test.go` keyword-set policy through AC-1.

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

3. **Keyword-set meta-coverage continues to pass.** `m0158_ac5_meta_coverage_test.go` remains in place through AC-1, AC-2, AC-3; the keyword-set entries for the 9 dropped cells are removed alongside the drop so the meta-test stays green. AC-4 removes the keyword-set file entirely once the bijection meta-test lands.

4. **Sabotage-verifiable.** Re-adding a dropped cell to `branch.Rules()` fires the absence subtest; removing a retained cell fires the presence subtest. The discriminating tests fire either way.

**Edge cases:**

- **M-0161-era cells stay registered.** AC-1 drops only M-0158-era doc-only cells; the 1-cell-per-AC chokepoints M-0161 added (`branch-cell-isolation-escape-oracle-failure`, `-shallow-clone`, `-orphaned-ai-commit`, `-rename-survival`, `-id-rename-untrailered`, `-detached-head-preflight`, `-promote-on-wrong-branch`) all stay — they're part of the M-0161/AC-6 drift-policy chokepoint surface.
- **Meta-coverage transition.** Between AC-1 and AC-4, `m0158_ac5_meta_coverage_test.go` is the active meta-coverage; the keyword-set entries for the 9 dropped cells need removal in the same AC-1 commit so the meta-test stays green. AC-4 deletes the file entirely.

**References.**

- M-0161/AC-9 body §"Part 1" — the inherited drop list this AC discharges
- [M-0158](M-0158-layer-4-branch-choreography-spec-cells-drift-policy-extension.md) — the catalog whose doc-only cells this AC drops
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap this AC partially addresses (closes G-0210 once AC-2..AC-4 land)
- `internal/workflows/spec/branch/rules.go` — the catalog the AC touches
- `internal/policies/m0158_ac5_meta_coverage_test.go` — keyword-set meta-coverage that stays in place through AC-3, removed in AC-4

### AC-2 — M-0161 cell expansion: organic count via bijection invariants

**Observable behavior.** The branch-choreography catalog at `internal/workflows/spec/branch/rules.go` is expanded with one cell per discriminating E2E subtest across the M-0161/AC-1..AC-8 surfaces. The exact cell count is determined organically by the test surface — the deliverable is bijection-invariant correctness (every E2E subtest pins exactly one cell; every cell carries exactly one Pin), not arithmetic matching to the M-0161/AC-9 body's "66 new cells" forecast.

The M-0161/AC-9 body's "76 total" estimate stays as a forecast; the actual count is reported at AC-4 wrap. AC-4's bijection meta-test asserts the invariants, not cardinality.

**Cells added (organic count, ~57-77 depending on subtest discrimination):**

The M-0161 AC bodies define the matrix shapes:

- M-0161/AC-1 — 4 trunk-name shapes (TestAuthorize_AC1_NonMainTrunkNames_Accept subtests)
- M-0161/AC-2 — 16 rung-pair cells + 1 override (TestAuthorize_AC2_RungPair_Matrix subtests)
- M-0161/AC-3 — 13 oracle-state subtests (TestBranchOracle_AC3_OracleErrors_Matrix paired) + 2 sovereign-override subtests
- M-0161/AC-4 — 11 shallow-clone subtests + 2 sovereign-override subtests
- M-0161/AC-5 — 7 force-push-orphan subtests + 1 cell-7 reflog-disabled composition subtest (cell-5 deferred per D-0020)
- M-0161/AC-6 — 9 rename-resolution subtests
- M-0161/AC-7 — 7 detached-HEAD subtests (B1 follow-up included)
- M-0161/AC-8 — 8 promote-on-wrong-branch subtests (cell-6 detached-HEAD deferred per AC-8 body)

Each subtest gets exactly one Pin call to exactly one cell. AC-3 (Pin registry) provides the call surface; AC-2 wires the cell entries; AC-4 enforces the bijection.

**Mechanical assertions:**

1. **Cell-presence verification.** A test under `internal/policies/m0162_ac2_expanded_set_test.go` asserts each E2E test function's expected cell IDs are present in `branch.Rules()`. The test parses the E2E files for Pin call sites (per AC-3's registry) and matches them against `branch.Rules()` entries.

2. **Subtest-to-cell mapping.** Every E2E subtest under `internal/cli/integration/branch_scenarios_*.go`, `isolation_escape_*.go`, `detached_head_*.go`, and `promote_wrong_branch_*.go` calls `branchtest.Pin(cellID, t.Name())` at setup (AC-3 prerequisite). AC-2's cell-set must cover every Pin call site.

3. **Sabotage-verifiable.** Removing a cell that an E2E subtest references makes the cell-presence test fail naming the orphan cell; adding a cell without a Pin call from any subtest fires the AC-4 bijection invariant (post-AC-4) — AC-2's own discriminator is the AC-1 paired-test invariant inherited from M-0158/AC-5.

4. **Catalog count reported, not pinned.** The wrap report records the actual cell count after expansion. The M-0161/AC-9 body's "76 total" forecast is a planning estimate, not a contract.

**Edge cases:**

- **AC-3 prerequisite.** AC-2's cell additions are useless without Pin call sites; AC-3 must land first (or co-land) so the Pin registry exists. Per the foundation-up sequencing (M-0162 §Scope decision), AC-2 ships its cells with Pin call additions to existing E2Es as a single AC-2 deliverable. AC-3 (Pin registry) is the AC-2 prerequisite — if AC-3 is sequenced strictly after AC-2 per the locked ordering, AC-2 ships cells WITHOUT Pin calls and AC-3 adds the Pin calls. **Decision: AC-2 ships cells only; AC-3 ships Pin registry + every E2E's Pin call addition. The bijection invariants pass at AC-4.**
- **M-0159 framework subtests.** Some matrix-level tests (M-0159's `RunScenarios([]Scenario{...})`) produce per-row subtests via `t.Run`. The Pin call goes inside the Scenario's Setup function so each subtest pins its own cell. AC-3's API supports `Pin(cellID, t.Name())` inside the closure.
- **M-0161/AC-5 cell 5 + M-0161/AC-8 cell 6 deferrals.** The deferred cells per D-0020 and AC-8 body's deferral are NOT expanded; their absent test functions mean no Pin call → no orphan finding. AC-4's bijection invariant tolerates the gaps because the cells genuinely don't exist (not "registered but unpinned").

**References.**

- M-0161/AC-9 body §"Part 2" — the inherited expansion scope this AC discharges
- M-0161 AC bodies — the matrix shapes that determine the per-AC cell counts
- AC-3 (this milestone) — the Pin registry prerequisite for the Pin call surface
- AC-4 (this milestone) — the bijection meta-test that validates AC-2's expansion correctness
- `internal/cli/integration/branch_scenarios_*.go` + sibling files — the E2E surface AC-2 references

### AC-3 — branchcell.Pin registry under //go:build testpins + branchtest sub-package

**Observable behavior.** A new test-only package `internal/workflows/spec/branch/branchtest` introduces a `Pin(cellID, testFunctionName string)` registry callable from any test under the `//go:build testpins` build tag. The registry accumulates pins for later inspection by AC-4's bijection meta-test.

The package + its single source file `pin.go` carry the `//go:build testpins` header so production `go build` omits both. CI runs and the Makefile's `test-pins` target carry `-tags testpins`; bare `go test ./...` without the tag silently skips the pin-calling tests and the bijection meta-test (the latter also tagged). This is the deliberate trade-off: the registry is opt-in by tag rather than always-on; newcomers running tests locally either use the Makefile or learn the tag.

**Per the M-0162 Q&A decision §"Pin shape" (locked at AC-body-authoring time):** option 1 (`//go:build testpins + dedicated branchtest sub-package`) was selected over the AC-9 body's `_test_helpers.go` alternative (which was found incorrect — that suffix doesn't actually keep files out of production). The branchtest sub-package gives the test-only nature an import-path-level marker AND the build tag enforces link-time exclusion.

**API shape:**

```go
//go:build testpins

package branchtest

// Pin records that a test function exercises a specific
// branch.Rules() cell. Calls accumulate into a process-local
// registry inspected by the bijection meta-test at AC-4.
//
// Calls from tests inside `t.Run` should pass t.Name() so the
// subtest's full name (TestX/sub-row) appears in the registry.
//
// Calls outside the testpins build tag are link-time errors;
// the registry is never present in production binaries.
func Pin(cellID, testName string) { ... }

// Pins returns a snapshot of accumulated pins. Used by the
// bijection meta-test at internal/policies/.
func Pins() map[string][]string { ... }
```

**Pin call sites added by AC-3:**

Every E2E subtest under:
- `internal/cli/integration/branch_scenarios_ac4_test.go` (M-0159/AC-4 ack scenarios)
- `internal/cli/integration/branch_scenarios_ac5_test.go` (M-0159/AC-5 trailer-verb-unknown)
- `internal/cli/integration/branch_scenarios_ac6_test.go` (M-0159/AC-6 cherry-pick)
- `internal/cli/integration/isolation_escape_oracle_scenarios_test.go` (M-0161/AC-3)
- `internal/cli/integration/isolation_escape_shallow_scenarios_test.go` (M-0161/AC-4)
- `internal/cli/integration/isolation_escape_force_push_scenarios_test.go` (M-0161/AC-5)
- `internal/cli/integration/isolation_escape_rename_scenarios_test.go` (M-0161/AC-6)
- `internal/cli/integration/detached_head_scenarios_test.go` (M-0161/AC-7)
- `internal/cli/integration/promote_wrong_branch_scenarios_test.go` (M-0161/AC-8)
- `internal/cli/integration/authorize_scenarios_test.go` (M-0161/AC-1 + AC-2)

The Pin call goes inside each Scenario's `Setup` function (for `RunScenarios` framework consumers) or inside each subtest's body (for direct `TestX` functions).

**Mechanical assertions:**

1. **Build-tag exclusion.** A test under `internal/policies/m0162_ac3_build_tag_test.go` builds the production `aiwf` binary without `-tags testpins` and asserts the `branchtest` package symbols are NOT present in the resulting binary (via `go tool nm` or equivalent). Sabotage-verified by removing the build-tag header.

2. **API existence verification.** A test under the testpins tag asserts `Pin()` accepts the two-string signature and `Pins()` returns the accumulated map shape. Catches a future refactor that changes the API.

3. **Pin-call presence in every E2E.** A test parses every file under `internal/cli/integration/` matching the AC-3 surface list above for `branchtest.Pin(...)` call sites. Each file must have at least one Pin call (per AC-4 invariant #1 inherited). Sabotage-verified by removing a Pin from an E2E.

4. **Sabotage-verifiable.** Removing the build-tag header fires the build-tag exclusion test (symbols appear in production); removing a Pin from an E2E fires the presence test naming the missing call site; removing the registry's accumulation behavior fires AC-4's bijection invariants.

**Edge cases:**

- **Pin call inside parallel subtests.** Subtests calling `t.Parallel()` run concurrently; the Pin registry needs a `sync.Mutex` around the accumulator to be data-race-free. The test-only nature means a sync.Mutex import in test code is acceptable.
- **Pin call from inside `Setup` vs `t.Run` body.** Both shapes supported by passing `t.Name()` explicitly. The M-0159 RunScenarios framework calls `Setup` after entering the subtest's `t.Run` so `t.Name()` resolves to the subtest's full path.
- **Newcomer running bare `go test`.** Without the `testpins` tag, the registry is empty and the bijection meta-test (also tagged) is skipped. CI and the Makefile carry the tag; local newcomers see no pin-related output. Documented in `internal/workflows/spec/branch/README.md` (new file at AC-3 time).

**References.**

- M-0161/AC-9 body §"Part 3" — the inherited Pin registry scope this AC discharges
- M-0162 Q&A §"Pin shape" — the build-tag + branchtest sub-package decision
- AC-4 (this milestone) — the bijection meta-test that consumes the Pin registry
- `internal/cli/integration/` — the E2E surface AC-3 wires Pin calls into

### AC-4 — Bijection meta-test replaces M-0158/AC-5 keyword-set; 4 invariants

**Observable behavior.** A new bijection meta-test at `internal/policies/branch_cell_bijection_test.go` (under `//go:build testpins`) enforces four invariants between `branch.Rules()` and the `branchtest.Pins()` registry. The existing keyword-set meta-coverage at `internal/policies/m0158_ac5_meta_coverage_test.go` is removed in the same commit; the bijection meta-test pins a strictly stronger claim than the keyword-set's ≥1 match per AC-9 body lines 634-640.

Three meta-cells are registered for the bijection invariants themselves so the catalog records its own enforcement chokepoints alongside the rule chokepoints.

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

**Drift policy extension:**

- The existing M-0158 drift policy at `internal/policies/m0158_ac6_drift_test.go` continues to assert "every ClassBranchChoreography code referenced by an Illegal cell"; AC-4 leaves it alone.
- A new policy at `internal/policies/m0162_ac4_drift_test.go` asserts the bijection holds at CI time (consumed by every CI run via the `testpins` tag). Adding a cell to `branch.Rules()` without a Pin OR adding a Pin without a matching cell fails CI.

**Mechanical assertions:**

1. **Four-invariant bijection meta-test.** The four subtests above. Each fails on its specific sabotage; CI runs them under `-tags testpins`.

2. **Keyword-set deletion verification.** A structural test asserts `internal/policies/m0158_ac5_meta_coverage_test.go` does NOT exist. Catches a future change that re-adds the file (e.g., a confused merge).

3. **Meta-cell registration verification.** A test asserts the 3 meta-cells exist in `branch.Rules()` and each has at least one Pin (closing the meta-coverage loop: the bijection enforcer is itself a Pinned cell).

4. **Sabotage discrimination.** Each of the 4 invariants has a paired sabotage test that constructs a fixture violating the invariant and asserts the production invariant test fires. The sabotage tests are themselves tagged `testpins`.

5. **CI tag verification.** The CI workflow (`.github/workflows/go.yml`) is updated to add `-tags testpins` to the test step. Without it, the bijection meta-test silently skips; with it, the bijection invariants are enforced.

**Edge cases:**

- **AC-3 prerequisite (Pin registry exists).** AC-4 cannot land without AC-3's registry being available. Per the foundation-up ordering, AC-3 lands first.
- **AC-2 prerequisite (cells expanded).** AC-4 enforces the bijection over AC-2's expanded catalog. If AC-2 ships an under-expanded catalog (some E2E subtests without paired cells), AC-4's invariant #2 (orphan-Pin detection) fires at CI time and AC-2 returns for additional cells. This is the discipline working as designed.
- **M-0161/AC-5 cell-5 + M-0161/AC-8 cell-6 deferrals.** Both have no test function (the deferred cells genuinely don't exist). AC-4's invariants tolerate the gap because neither the cell nor the Pin exists — invariant #1 doesn't fire (no cell to find unpinned); invariant #2 doesn't fire (no orphan Pin); invariants #3 and #4 don't apply. The deferrals stay deferrals; AC-4 doesn't force the deferred scope.
- **Test-time pin accumulation race.** The Pin registry's mutex (AC-3) ensures the accumulator is data-race-free under `t.Parallel`. AC-4's meta-test reads the accumulator after all tests in the package have completed (via `TestMain` ordering or a final-stage test); reading is safe because no Pin calls are in flight.

**References.**

- M-0161/AC-9 body §"Part 4" + §"Part 5" — the inherited bijection scope this AC discharges
- AC-2 + AC-3 (this milestone) — the cell catalog + Pin registry AC-4 enforces invariants over
- `internal/policies/m0158_ac5_meta_coverage_test.go` — the keyword-set file AC-4 deletes
- `internal/policies/m0158_ac6_drift_test.go` — the existing drift policy AC-4 leaves intact
- [G-0210](../../gaps/G-0210-m-0158-spec-table-contains-9-documentation-only-or-duplicate-cells.md) — the gap this AC closes (full closure when AC-1, AC-2, AC-3, AC-4 all land)

