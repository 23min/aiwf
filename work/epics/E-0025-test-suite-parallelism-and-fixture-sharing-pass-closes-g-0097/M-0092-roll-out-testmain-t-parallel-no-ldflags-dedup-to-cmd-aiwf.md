---
id: M-0092
title: Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/
status: in_progress
parent: E-0025
depends_on:
    - M-0091
tdd: none
acs:
    - id: AC-1
      title: cmd/aiwf/setup_test.go lands with TestMain setting GIT identity
      status: met
    - id: AC-2
      title: per-file audit produces a serial skip-list; safe tests gain t.Parallel
      status: met
    - id: AC-3
      title: binary_integration_test.go shares the no-ldflags build via sync.Once
      status: met
    - id: AC-4
      title: go test -race -parallel 8 ./cmd/... reliable across 10 runs
      status: deferred
---

# M-0092 — Roll out TestMain + t.Parallel + no-ldflags dedup to cmd/aiwf/

## Goal

Apply the `TestMain` + `t.Parallel()` pattern across `cmd/aiwf/` tests and share the no-ldflags binary build via `sync.Once` in `cmd/aiwf/binary_integration_test.go`. After this milestone, `go test -race -parallel 8 ./cmd/...` is reliable across 10 consecutive runs and the binary build cost is paid once per test binary instead of five times.

## Context

`cmd/aiwf/` tests have more parallelism nuance than `internal/*`: some intentionally rely on subprocess isolation (`runBin`, the existing `aiwfBinary` helper), some mutate process-level state (env, `os.Args`), and `integration_g37_test.go`'s 11 bare-origin + N-clone tests have dense fan-out. The right shape is per-file audit, not blind application — but the underlying pattern (TestMain for env; t.Parallel where safe) is the same as M-0091's.

The `binary_integration_test.go` dedup is a separate, sympathetic optimization: 5 of 7 tests in that file build a non-stamped binary, paying ~1 second per test on first build. Sharing the build via `sync.Once` (matching the `aiwfBinary` precedent in `integration_test.go`) saves 4 builds per run. The 2 ldflags-stamped tests still build their own.

This milestone is single-commit per the epic's Constraints. M-0091 relaxes that rule for itself because it's per-package refactor; `cmd/aiwf/` is one package, so one commit is the natural shape.

## Acceptance criteria

(ACs allocated via `aiwf add ac`; bodies follow below.)

## Constraints

- **One commit for the milestone**, carrying `aiwf-verb`, `aiwf-entity: M-0092`, and `aiwf-actor` trailers.
- **Subprocess-isolating tests stay subprocess-isolating.** This milestone does not move any `runBin` caller to in-process `run([]string{...})`. That conversion is explicitly out per the epic — subprocess isolation is load-bearing for exit-code / stdout-stderr / env-isolation assertions and the kernel's "test the actual binary" stance.
- **Topology sharing across `integration_g37_test.go` stays deferred.** Each test's bare-origin + N-clone setup has enough variation that sharing is real refactor work. Plain `t.Parallel` adoption (with the dense-fan-out caveat in the skip-list) is the right scope here.
- **No test semantics change.** Same `-race -count=10` discipline as M-0091's AC for cross-run reliability.

## Design notes

- The `binary_integration_test.go` dedup uses the same `sync.Once` pattern already living in `cmd/aiwf/integration_test.go::aiwfBinary` — copy the shape, don't reinvent.
- `integration_g37_test.go`: lean toward keeping it serial *unless* a quick audit shows the bare-origin/clone setup actually parcels cleanly. If parcelling looks risky, the whole file goes on the serial skip-list with a one-line rationale and the deferral is captured in Reviewer notes — not a Deferral entity, because the topology-sharing question is already deferred at the epic level.

## Surfaces touched

- `cmd/aiwf/setup_test.go` (new)
- `cmd/aiwf/binary_integration_test.go` — `sync.Once` shared build
- Every `cmd/aiwf/*_test.go` — `t.Parallel()` adoption per the audit

## Out of scope

- Moving `runBin` callers to in-process `run([]string{...})`.
- Topology sharing across `integration_g37_test.go`.
- Pre-baked `aiwf init`-ed skeleton tempdir snapshot.
- The `CLAUDE.md ## Test discipline` section and the `setup_test.go`-presence policy test (M-0093).

## Dependencies

- **M-0091** — internal/* rollout. The pattern, the cap, and the spike-reference shape all come from there. The cap (`-parallel 8` in Makefile + workflows) is the load-bearing prerequisite: this milestone runs its `-race -parallel 8` validation against that cap.

## References

- E-0025 epic spec.
- M-0091 — the pattern source for this milestone.
- `cmd/aiwf/integration_test.go::aiwfBinary` — `sync.Once` precedent.
- `internal/verb/setup_test.go` — `TestMain` reference (via M-0091, in turn from the spike).

## Work log

### AC-1 — TestMain + helpers neutralized (in the wrap commit, `a7fcd25`)

`cmd/aiwf/setup_test.go` introduced with a uniform TestMain that `os.Setenv`s the 4 GIT identity vars. Two existing helpers had their `t.Setenv` blocks stripped (now redundant under TestMain, and `t.Setenv` panics under `t.Parallel`): `setupCLITestRepo` in `main_test.go` and `initTrailerRepo` in `canonicalize_history_test.go`. The file's comment block carries the full skip-list — that lands as part of AC-2's audit.

### AC-2 — t.Parallel adoption across cmd/aiwf/ (in the wrap commit)

447 `func Test*(t *testing.T)` audited across ~76 test files. **337 gained `t.Parallel()`** as their first statement; **110 stay serial** across four categories, fully documented in `setup_test.go`'s skip-list comment:

- `integration_g37_test.go` (whole file, 11 tests) — dense bare-origin + N-clone subprocess fan-out; topology-sharing deferred at the epic level.
- ~70 tests calling `captureStdout` / `captureStderr` / `captureRun` (across ~20 files) — the helpers mutate package-level `os.Stdout` / `os.Stderr` (a goroutine-shared fd), incompatible with `t.Parallel`. Helpers themselves are not refactored under this milestone (per constraint "no test-semantics change").
- 13 tests calling `t.Setenv` directly (actor_test.go, doctor_cmd_test.go) — panic under `t.Parallel` by stdlib design.
- 6 tests calling `os.Chdir` (completion_helpers_test.go, whoami_cmd_test.go) — process-wide cwd mutation races.

Five tests opted their table-driven `t.Run` subtests into a second nested `t.Parallel()` where the per-iteration fixtures are independent: `TestActorPattern`, `TestResolveActor_ExplicitInvalid`, `TestReorderFlagsFirst`, `TestStripTrailers`, `TestHistory_NarrowTrailerMatchesCanonicalQuery`.

One mid-audit catch: an early classifier missed `captureRun` (a third stdout-mutating helper in `upgrade_cmd_test.go`). The race detector flagged it at `-count=3`; fixed by adding `captureRun` to the audit pattern and moving its 10 callers back onto the serial skip-list.

### AC-3 — sync.Once dedup for the no-ldflags binary build (in the wrap commit)

`cmd/aiwf/integration_test.go::aiwfBinary` already exists and shares a no-ldflags build via `sync.Once`. The implementation re-uses it: the 6 no-ldflags `buildBinary(t, tmp /* no ldflags */)` calls in `binary_integration_test.go` were swapped to `aiwfBinary(t)`. The 1 ldflags-stamped test (`TestBinary_VersionVerb_DerivesFromGoldFlags`) keeps `buildBinary` so its `-X main.Version=` stamping stays test-local. Four tests had their now-unused `tmp := t.TempDir()` removed.

### AC-4 — DEFERRED (see G-0125)

Three 10-run loops on the macOS dev host produced 7/10 to 8/10 passes with two flake modes:

1. `os/exec` deadlock inside `gitops.StagedPaths` (multiple different tests across runs — `TestRender_ProvenanceTabShowsAuthorizeScope`, `TestArchive_PerKindStorageLayout`, `TestRetitle*`, `TestRender_WellFormed*`, others). All stuck on a `git diff --staged` subprocess. The agent flagged this class up front as "macOS subprocess-stress, system-level not test-logic"; the per-test stack confirms — no single test is the cause.
2. Repo-lock collision (1/30): `aiwf add: another aiwf process is running on this repo`. Path that produced it not yet identified; per-test repos via `t.TempDir()` should prevent it.

Multiple tests participate; the cap-at-8 from M-0091/AC-1 fits internal/* but is too loose for cmd/aiwf/'s subprocess-heavy workload. CI Linux behavior is unknown until the milestone branch is pushed. Deferred to G-0125 with four remediation paths sketched (token-bucket around git invocations, lower per-package cap, refactor specific patterns, or accept macOS as degraded host). Decision deferred pending CI signal.

## Decisions made during implementation

- **AC-4 deferred to G-0125.** The strict "0 flakes across 10 runs" reading of AC-4 isn't reachable on macOS dev hosts at `-parallel 8` under cmd/aiwf/'s subprocess-heavy workload. Rather than paper over (CLAUDE.md "don't paper over a test failure"), the milestone ships the three behavioral ACs and defers the reliability bar to a gap that names the conditions, the observed flake modes, and the four candidate remediations. CI Linux signal is the next data point; if CI is reliably green, remediation can wait until a real consumer reports friction.
- **Helper refactor stays scoped.** The `captureStdout` / `captureStderr` / `captureRun` helpers mutate package-level `os.Stdout` / `os.Stderr`, making every caller incompatible with `t.Parallel`. Refactoring them to per-test pipes would unlock another ~70 tests, but per the M-0092 constraint set "no test-semantics change," that conversion stays out and the 70 tests land on the serial skip-list. Candidate for a future gap if CI signal motivates it.

## Validation

### Build + lint + check

- `go build ./cmd/aiwf` — green.
- `golangci-lint run` — 0 issues.
- `aiwf check` — 0 errors on this branch.

### AC-4 attempted (10-run `-race -parallel 8 -count=1` on cmd/aiwf/)

Three 10-run loops on the macOS dev host (warm Go build cache, machine cleaned of stale `aiwf.test` processes between loops):

| Loop | Passes | Wall (avg) | Flake modes |
|---|---|---|---|
| 1 (initial) | 7/10 | 86–96s when passing, 661s when timing out | 1 lock-collision + 2 timeouts |
| 2 (after process cleanup) | 8/10 | 84–105s when passing, 331s when timing out | 1 lock-collision + 1 timeout |
| 3 (with `-v` for diagnostics) | 7/10 | 79–94s when passing, 331s when timing out | 3 timeouts |

Aggregate: **22/30 passes** on a single dev machine. Multiple tests participate in the timeouts; no single fixable culprit. See `## Reviewer notes` for the captured failure shape and the AC-4 deferral context.

### Wall-time speedup (AC-2 / AC-3 aggregate, single iteration when clean)

| | Wall time |
|---|---|
| Baseline (pre-conversion) | 174s |
| Post-conversion (clean run) | 87s |
| **Speedup** | **~47%** |

Per-iteration speedup is the same ~2× shape as M-0091 — the wins compound, not stack, but cmd/aiwf was a meaningful chunk of total CI time.

## Deferrals

- **AC-4 → G-0125** "cmd/aiwf -race -parallel 8 flakes under subprocess fan-out (macOS)". The gap captures three 10-run loop results, the two flake modes (`os/exec` deadlock + repo-lock collision), the systemic shape (no single test is the cause), and four remediation paths. Resolution awaits CI Linux signal post-push.

## Reviewer notes

- **AC-4 deferral is the headline trade-off.** Three behavioral ACs (TestMain + skip-list, broad t.Parallel adoption, no-ldflags build dedup) all met. The fourth — the reliability *measurement* under -race -parallel 8 — could not pass strict-reading 0/10-flakes on the macOS dev host. The agent flagged this class at AC-2 close-out; iterative loops + `-v` diagnostics confirmed the systemic nature. Deferring to G-0125 keeps the discipline ("don't paper over") and the audit trail; the gap names what would need to change to ratify AC-4 (token-bucket, lower per-package cap, refactor, or accept macOS as degraded host).
- **captureRun catch.** The audit's first pass missed a third stdout-mutating helper (`captureRun` in upgrade_cmd_test.go). The race detector caught it at `-count=3`; the fix added 10 upgrade tests to the serial skip-list. Worth reading the setup_test.go skip-list for the full taxonomy — a future helper refactor (per-test pipes instead of package-level mutation) would unlock those 70 tests in one move.
- **Subprocess-isolating tests unchanged.** `runBin` callers, integration_g37_test.go's bare-origin/clone topology, and the in-process dispatcher's `setupCLITestRepo` discipline (`--skip-hook` to avoid the test binary firing as a hook) all stay intact. The milestone explicitly stays out of those conversions per the epic.
- **Stale-process cleanup.** During the AC-4 investigation, two zombie `aiwf.test` processes from previous timeout-killed iterations were found running for 28+ minutes, eating CPU and skewing later runs. The 10-run script in subsequent loops `pkill -f "aiwf.test "` between iterations to prevent zombie accumulation. The G-0125 body documents this as a local dev workflow; CI doesn't need it (fresh process per workflow).
- **M-0093's lock-in still pending.** The setup_test.go convention is reviewer-enforced until M-0093 lands the `internal/policies/` chokepoint asserting every cmd/aiwf-style test-bearing package has a setup_test.go.

### AC-1 — cmd/aiwf/setup_test.go lands with TestMain setting GIT identity

A new `setup_test.go` in `cmd/aiwf/` declares a `TestMain` that calls `os.Setenv` for `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL` before `m.Run()`. The shape matches `internal/verb/setup_test.go`. Tests in the package no longer call `t.Setenv` for these four variables except where they deliberately clear them.

### AC-2 — per-file audit produces a serial skip-list; safe tests gain t.Parallel

Every `*_test.go` file in `cmd/aiwf/` is audited: which tests can safely `t.Parallel()` and which must stay serial. Stay-serial criteria: calls `t.Setenv`/`t.Chdir`, mutates `os.Args`, depends on a package-level var another test could clobber, or saturates a shared subprocess limit (the `integration_g37_test.go` cluster is the canonical case — its 11 tests each spin up a bare origin + multiple clones; running them all in parallel risks file-descriptor or process-table exhaustion). The skip-list lives as a `// Serial tests: …` comment in `setup_test.go` with one-line rationales. Every test not on the skip-list calls `t.Parallel()` as its first non-`t.Helper()` statement.

### AC-3 — binary_integration_test.go shares the no-ldflags build via sync.Once

A package-private helper modelled on `aiwfBinary` in `cmd/aiwf/integration_test.go` builds the no-ldflags binary once on first call and returns the path on subsequent calls. The 5 tests that build a non-stamped binary call the new helper; the 2 ldflags-stamped tests build their own as today. Test count and assertions are unchanged.

### AC-4 — go test -race -parallel 8 ./cmd/... reliable across 10 runs

After the conversion commit lands, the milestone records a 10-run loop of `go test -race -parallel 8 ./cmd/...` with zero flakes and zero timeouts. Pasted into Validation at wrap. Flakes are root-caused, not papered over.

