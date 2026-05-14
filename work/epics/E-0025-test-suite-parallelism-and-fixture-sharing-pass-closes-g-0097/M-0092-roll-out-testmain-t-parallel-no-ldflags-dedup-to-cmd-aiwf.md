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
      status: open
    - id: AC-2
      title: per-file audit produces a serial skip-list; safe tests gain t.Parallel
      status: open
    - id: AC-3
      title: binary_integration_test.go shares the no-ldflags build via sync.Once
      status: open
    - id: AC-4
      title: go test -race -parallel 8 ./cmd/... reliable across 10 runs
      status: open
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

(filled during implementation)

## Decisions made during implementation

- (none yet)

## Validation

(pasted at wrap: 10-run `go test -race -parallel 8 ./cmd/...` log; baseline vs. post-conversion wall time)

## Deferrals

- (none yet)

## Reviewer notes

- (none yet)

### AC-1 — cmd/aiwf/setup_test.go lands with TestMain setting GIT identity

A new `setup_test.go` in `cmd/aiwf/` declares a `TestMain` that calls `os.Setenv` for `GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL`, `GIT_COMMITTER_NAME`, `GIT_COMMITTER_EMAIL` before `m.Run()`. The shape matches `internal/verb/setup_test.go`. Tests in the package no longer call `t.Setenv` for these four variables except where they deliberately clear them.

### AC-2 — per-file audit produces a serial skip-list; safe tests gain t.Parallel

Every `*_test.go` file in `cmd/aiwf/` is audited: which tests can safely `t.Parallel()` and which must stay serial. Stay-serial criteria: calls `t.Setenv`/`t.Chdir`, mutates `os.Args`, depends on a package-level var another test could clobber, or saturates a shared subprocess limit (the `integration_g37_test.go` cluster is the canonical case — its 11 tests each spin up a bare origin + multiple clones; running them all in parallel risks file-descriptor or process-table exhaustion). The skip-list lives as a `// Serial tests: …` comment in `setup_test.go` with one-line rationales. Every test not on the skip-list calls `t.Parallel()` as its first non-`t.Helper()` statement.

### AC-3 — binary_integration_test.go shares the no-ldflags build via sync.Once

A package-private helper modelled on `aiwfBinary` in `cmd/aiwf/integration_test.go` builds the no-ldflags binary once on first call and returns the path on subsequent calls. The 5 tests that build a non-stamped binary call the new helper; the 2 ldflags-stamped tests build their own as today. Test count and assertions are unchanged.

### AC-4 — go test -race -parallel 8 ./cmd/... reliable across 10 runs

After the conversion commit lands, the milestone records a 10-run loop of `go test -race -parallel 8 ./cmd/...` with zero flakes and zero timeouts. Pasted into Validation at wrap. Flakes are root-caused, not papered over.

