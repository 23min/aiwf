---
id: M-0220
title: Re-fixture heavy real-tree check integration tests to synthetic fixtures
status: done
parent: E-0053
tdd: advisory
acs:
    - id: AC-1
      title: Real-tree check rendering test removed; collapse property pinned synthetically
      status: met
    - id: AC-2
      title: Check integration tests use the shared AiwfBinary, not per-test BuildBinary
      status: met
    - id: AC-3
      title: Measured integration-suite wall-time reduction recorded in Validation
      status: met
    - id: AC-4
      title: Sibling heavy real-tree check tests audited and re-fixtured or justified
      status: met
---
## Goal

Cut the `internal/cli/integration` suite wall-time — the test-suite critical
path — by re-fixturing the heavy `aiwf check` integration tests that run against
the full real kernel tree to use synthetic fixtures sized to the property under
test.

Measured (G-0330 / M-0218 review): the package is ~78s isolated at `-parallel 8`
and the critical path of `go test ./...` (~128s under CPU contention); `make ci`
and the CI test job run it **without `-short`**, so its cost is paid on every
run. One test — `TestBinary_CheckDefault_KernelTreeShortOutput` — is **34s (44%
of the wall)**: it runs `aiwf check` on the real ~5,500-commit kernel tree plus
its own redundant `BuildBinary`, purely to assert a rendering property (warnings
collapse to summary form; errors stay per-instance) that a synthetic fixture
with a few findings proves in <1s.

Removing that long pole is projected to drop the integration wall to ~45-55s
(~25-30s off every `make ci` / CI run) and improve effective parallelism
(measured 2.6x of the VM's ~4.86x — the pole drags the tail).

## Notes

Behavior-preserving test refactor; the rendering contract is the correctness
surface and must be preserved assertion-for-assertion.

- **No coverage lost.** The real-tree smoke is not unique to this test —
  `aiwf doctor --self-check` (CI selfcheck job) and the pre-push hook already
  drive `check` on real trees end-to-end. If a real-tree smoke is still wanted,
  keep one minimal `-short`-gated smoke, not the 34s assertion-heavy test.
- **Shared binary.** Use the shared `AiwfBinary` (sync.Once) instead of per-test
  `BuildBinary` where no custom `-ldflags` is needed (13 self-builds today; the
  version/ldflags tests legitimately self-build and stay).
- **Measurement protocol (the M-0218 lesson).** Report the
  `internal/cli/integration` package wall at `-parallel 8`, warm, isolated,
  before/after; the per-test top contributors; and the `go test ./...`
  critical-path delta. Pin the "done" threshold (>=25s off the isolated warm
  wall).
- **Scope.** The dominant lever is itself an `aiwf check` test — squarely inside
  E-0053's "make `aiwf check` fast" charter. Optimizing the integration
  package's non-check tests is explicitly out of scope (a separate effort if
  ever warranted).
- Wall-clock is recorded in `## Validation`, deliberately NOT asserted inside a
  test — env-dependent timing pins are flaky.

### AC-1 — Real-tree check rendering test removed; collapse property pinned synthetically

Removed `TestBinary_CheckDefault_KernelTreeShortOutput` (ran `aiwf check`
against the real ~5,500-commit kernel tree; 35.6s = 45% of the isolated
package wall) purely to assert a rendering property. Its warning-collapse
assertion — `^\S+: warning ` must match zero per-instance lines — migrated into
the synthetic `TestBinary_CheckDefault_SummarizesWarnings`. The property is
fixture-source-agnostic (the renderer treats a finding identically regardless
of which check produced it), so the `messy` fixture pins it in <1s. The regex's
teeth are preserved by the renamed
`TestWarningCollapse_StructuralAssertion_CatchesPerInstanceWarnings` meta-test;
the orphaned integration-local `repoRootForTest` locator was deleted. Evidence:
`SummarizesWarnings` + meta-test green; independent reviewer confirmed no
assertion was silently dropped.

### AC-2 — Check integration tests use the shared AiwfBinary, not per-test BuildBinary

Converted 7 check tests from per-test `testutil.BuildBinary(t, tmp)` to the
shared `testutil.AiwfBinary(t)` (sync.Once — one `go build` per test process):
5 in `check_summary_binary_test.go` and 2 in `check_trunk_rename_seam_test.go`.
The two `-ldflags`-stamped version tests in `binary_integration_test.go`
legitimately need a per-invocation build and stay on `BuildBinary`. Evidence:
tests green; contributes to the AC-3 wall-time reduction.

### AC-3 — Measured integration-suite wall-time reduction recorded in Validation

Measured; see `## Validation`. Headline: `go test -parallel 8 ./...` (the
`make ci` wall) 93s -> 70s = 23s (25%).

### AC-4 — Sibling heavy real-tree check tests audited and re-fixtured or justified

Audit complete; see `## Validation`. The removed kernel test was the only
real-tree check-*rendering* test; every other heavy real-tree/check sibling is
either already synthetic or justified.

## Work log

All four ACs landed in one cohesive implementation commit `5c8882a5`
(`test(integration): re-fixture real-tree check test to synthetic (M-0220)`) —
the change is a single behavior-preserving test refactor.

- **AC-1** — deleted the real-tree kernel test + orphaned locator; migrated the
  collapse assertion to the synthetic `SummarizesWarnings`; renamed/repointed
  the meta-test. `TestBinary_Check*` + `TestWarningCollapse*` green.
- **AC-2** — 7 check tests → shared `AiwfBinary`; 2 ldflags version tests
  retained on `BuildBinary`.
- **AC-3** — measurement recorded below.
- **AC-4** — real-tree check-test audit recorded below.

## Validation

Behavior-preserving test refactor. Whole module green before and after; no
production code touched (+33 / −120 across 3 test files).

Measurement protocol (per the M-0218 lesson): warm build cache, `-count=1`
(test cache off), `-parallel 8` (the `make test` / `make ci` cap), measured
both isolated and under `./...` contention, before/after via a scoped
`git stash` of the change.

| Metric | Before | After | Reduction |
|---|---|---|---|
| `go test -parallel 8 ./...` wall (the `make ci` number) | 93s | 70s | **23s (25%)** |
| `internal/cli/integration` under `./...` contention | 90.0s | 67.0s | 23s (26%) |
| `internal/cli/integration` isolated (`-parallel 8`, warm) | 78s | 60s | 18s (23%) |

`TestBinary_CheckDefault_KernelTreeShortOutput` was 35.6s (45% of the isolated
package wall) before removal; gone after. The next-heaviest test after removal
is `TestSeam_InitThenDoctorSelfCheck` (~5s).

**Projection vs actual.** The spec floated ≥25s off / 45–55s isolated. Actual
is 23s off the `make ci` wall and 18s isolated — below projection, recorded
honestly. The gap is expected: at `-parallel 8` the 35.6s pole overlapped ~17s
with the tail, so removing it freed 18–23s of *wall*, not the full 35s. The
full 35s of redundant CPU work (a 5,500-commit git-history walk every run) is
removed regardless, so on a lower-core machine or cold CI the wall saving
approaches 35s. The operator reviewed the 23s/25% result and confirmed closing
as done.

Suite green: `go test -count=1 -parallel 8 ./...` exit 0 both before and after
(whole module — no regressions). `golangci-lint run`: 0 issues. `aiwf check`:
0 errors.

**AC-4 audit — real-tree check-test inventory of `internal/cli/integration`:**
- `TestBinary_CheckDefault_KernelTreeShortOutput` — the only real-tree
  check-*rendering* test → **removed** (AC-1).
- `archive_kernel_migration_test.go` (`findKernelRoot` copies the real
  `work/` + `docs/adr/`) — the ADR-0004 historical-migration proof; uses
  `aiwf check` only as a post-sweep clean-assertion. The real tree is essential
  to its purpose, so re-fixturing would defeat it → **justified**, left as-is
  (also a non-check test, out of M-0220 scope; build untouched).
- `check_trunk_rename_seam_test.go` — synthetic git repos, no real-tree access
  → already synthetic; builds now shared (AC-2).
- doctor self-check tests (`TestSeam_InitThenDoctorSelfCheck`,
  `Test{Binary,Run}_DoctorSelfCheck_Passes`) — doctor tests, not check tests;
  run against synthetic init'd repos → out of scope.
- Every other `filepath.Join(root, "work", "epics", …)` in the package uses a
  synthetic `t.TempDir()` root (grep-verified; independent reviewer concurred).

**TDD note.** M-0220 is `tdd: advisory` but is a behavior-preserving refactor —
no red→green→refactor cycle applies (existing green tests were modified, not
driven from a failing test). The ACs are met on mechanical evidence (the green
suite + the recorded measurement/audit). The `acs-tdd-audit` advisory (×4, met
without a tracked phase) accurately reflects that no phases were tracked and is
expected; it is a warning, not an error.

## Decisions made during implementation

- **Re-fixture strategy: fold + delete (Option 1), not synthetic-`.git`
  re-fixture (Option 2).** The kernel test's rendering property is
  fixture-source-agnostic and already pinned by the synthetic
  `SummarizesWarnings` + the meta-test; rebuilding it on a small synthetic
  `.git` tree would exercise nothing the renderer distinguishes (an error is an
  error regardless of source check). AC-1 and AC-2 were reworded to match — the
  named real-tree test ceases to exist; AC-2 re-scopes to the surviving check
  tests. No ADR: this shapes the test suite, not the architecture.

## Deferrals

None within the milestone's scope. Observation recorded (deliberately **not**
filed as a gap): the suite's remaining floor (~60s isolated) is the genuine
cost of end-to-end seam testing (real-binary subprocess + real git commits),
not waste — the measured `TestRun_` (in-process) vs `TestBinary_` (shelled)
self-check pair (2.87s vs 3.13s) shows the subprocess adds only ~8%. A further
M-0220-style right-sizing audit could examine paired `TestRun_`/`TestBinary_`
redundancy (e.g. the two doctor self-check variants at ~3s each), but each
demotion trades seam coverage — measure-and-judge, not a free win. File a gap
only if that audit is chartered.

## Reviewer notes

- Test-only change; no production code touched.
- **Coverage-loss check (the key risk):** the deleted kernel test's assertions
  are a strict subset of the synthetic `SummarizesWarnings` (which asserts
  exact codes/counts/ordering/footer), and the real-tree check pipeline
  (git-history checks) remains covered by `doctor --self-check`, the synthetic
  trunk-rename tests, `provenance_check_test.go`, and `internal/check` unit
  tests. The deleted test was redundant assertion coverage, not a unique
  exercise of any production path.
- The migrated `warningPerInstanceRE` assertion is live, not vacuous: the
  `messy` fixture emits 9 warning instances across 6 codes (all collapsed), so
  "zero per-instance warning lines" is a real assertion against warning-bearing
  output; its teeth are independently proved by the meta-test.
- **Independent reviewer** (`reviewer` agent, fresh context, adversarial brief
  over the diff): **APPROVE** — all five load-bearing claims (behavior-preserving,
  no coverage lost, ldflags-test safety, audit completeness, no dead code)
  verified by direct inspection.

