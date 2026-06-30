---
id: M-0220
title: Re-fixture heavy real-tree check integration tests to synthetic fixtures
status: in_progress
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
      status: open
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

### AC-2 — Check integration tests use the shared AiwfBinary, not per-test BuildBinary

### AC-3 — Measured integration-suite wall-time reduction recorded in Validation

### AC-4 — Sibling heavy real-tree check tests audited and re-fixtured or justified

