---
id: M-0242
title: Fault injection via external observation
status: in_progress
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: A process killed while holding repolock releases it via kernel fd cleanup
      status: met
      tdd_phase: done
    - id: AC-2
      title: A process killed mid-write never leaves a half-written entity file
      status: met
      tdd_phase: done
    - id: AC-3
      title: Lock-held and temp-file states are detected with no production-code change
      status: met
      tdd_phase: done
    - id: AC-4
      title: A disk-full or permission-denied write surfaces as a clean error
      status: met
      tdd_phase: done
---

## Goal

Implement scenario tier 3: `kill -9` mid-verb and disk-level fault
injection, triggered by externally observable proxies rather than a hook
compiled into production code.

## Context

M-0240 shipped the skeleton; M-0241 proved out the loose/statistical race
mechanism. This milestone adds the harness's other orchestration primitive
— directed timing via external observation — for the one class of scenario
that needs to hit a specific instant (mid-lock-hold, mid-write) rather than
just "close together."

## Acceptance criteria

### AC-1 — A process killed while holding repolock releases it via kernel fd cleanup

The harness detects "another process holds the lock" from outside, via a
non-blocking `flock` probe against the same lockfile — no code change to
`internal/repolock`. Once detected, it sends `SIGKILL` to the lock-holding
process and confirms a subsequent `Acquire` succeeds immediately, proving
the kernel's fd-cleanup-on-exit behavior `repolock`'s own doc comment
already claims.

### AC-2 — A process killed mid-write never leaves a half-written entity file

The harness detects "a write is in flight" by watching for
`pathutil.AtomicWriteFile`'s sibling temp file to appear (via `fsnotify` or
polling) — again, no production-code change. Killing the process at that
instant and inspecting the target file afterward confirms it's either
fully-old or fully-new, never partially written.

### AC-3 — Lock-held and temp-file states are detected with no production-code change

Both AC-1 and AC-2's detection mechanisms are asserted, on their own, to
require zero changes to `internal/repolock` or `internal/pathutil` — the
harness observes existing, unmodified side effects (the lockfile's flock
state, the temp file's transient existence), it doesn't instrument them.

### AC-4 — A disk-full or permission-denied write surfaces as a clean error

A write that fails partway due to `ENOSPC` or `EACCES` (simulated via a
small disk quota / restricted-permission fixture directory) produces a
clean, wrapped error — not a corrupted file, not a panic.

## Constraints

- No failpoint or pause-hook mechanism in production code — external
  observation only, per this epic's constraint. If a specific fault-timing
  need genuinely can't be met this way, that's an escalation to discuss,
  not a silent workaround.
- A random-delay-then-kill fallback is acceptable only where no external
  proxy exists for the target window; document which scenarios (if any)
  use it and why.

## Design notes

- This is the milestone where the epic's "no failpoints by default"
  constraint gets its first real test — if AC-1/AC-2's external-observation
  approach turns out insufficient for some case, that's a decision to
  surface explicitly (per the epic's Out of scope on failpoints), not to
  quietly work around.

## Surfaces touched

- `internal/stresstest/` (new scenario files; a small external lock-probe /
  temp-file-watch helper)

## Out of scope

- The named G-0212/G-0269 scenarios — M-0243, even though some of them
  (e.g. force-amend) involve process-level timing too; kept separate so
  this milestone stays focused on the fault-injection *mechanism*, not a
  specific named scenario.
- Any change to `repolock`/`pathutil` production code.

## Dependencies

- M-0240 — the harness skeleton.

## References

- `docs/initiatives/robustness-correctness-stress-testing.md`
- `internal/repolock/repolock_unix.go`

---

## Work log

### AC-1 — repolock kill -9 releases via kernel fd cleanup

Built a small `internal/stresstest/lockholder` helper (a nested `package
main`, no change to `internal/repolock` itself) that acquires the repo
lock and blocks until killed. `LockKillScenario` launches it as a real
subprocess, confirms externally that the lock reads as held via
`repolock.Acquire(dir, 0)` — repolock's own pre-existing zero-timeout
probe mode — SIGKILLs the holder, and confirms an immediate re-acquire
succeeds. 24 new tests (pure classify/decision-logic tables plus
real-binary integration tests covering every branch, including the
ready-timeout and cannot-acquire paths); a 6-mutation vacuity probe
confirmed every decision branch actually catches a regression · commits
6a287690, 5008a006 · tests 24/24.

### AC-2 — kill mid-write never leaves half-written entity file

`MidWriteKillScenario` drives a twin control/target repo pair seeded with a
large-bodied (10MB, empirically calibrated for a comfortably-wide temp-file
window) gap entity. The target repo's real `aiwf promote` is watched from
outside for `pathutil.AtomicWriteFile`'s sibling `.aiwf-tmp-*` file — its
own already-documented naming convention, no pathutil code change, per
AC-3 — and SIGKILLed the instant it appears; the oracle asserts the entity
file afterward is byte-identical to either the pre-write or fully-written
state, never a third value. Discovered and filed G-0391 (a mutating verb's
lock-busy refusal bypasses `--format=json` entirely) along the way; the
scenario and its tests work around it rather than depending on it. 14 new
tests (pure classify table plus real-binary integration covering every
reachable branch); a 5-mutation vacuity probe confirmed every decision
branch actually catches a regression · commits 237475a3, e14c36d3 · tests
14/14.

### AC-3 — detection mechanisms require zero repolock/pathutil changes

`TestNoNewExportsInRepolockOrPathutil` parses `internal/repolock/repolock_unix.go`
and `internal/pathutil/{pathutil,atomic}.go` via `go/parser` and asserts their
exported top-level surface is exactly the pre-existing set AC-1's and AC-2's
probes depend on — `Acquire`/`ErrBusy`/`Lock`/`Lock.Release` for repolock;
nothing at all for pathutil, since AC-2 only globs for its already-documented
`.aiwf-tmp-*` naming convention and never imports the package. A future edit
needing a new exported symbol to make either probe work would grow this set
and fail here. 1 new test (a 3-case table); a 4-mutation vacuity probe
confirmed every parsing branch actually catches a regression · commit
dc47799b · tests 1/1.

### AC-4 — disk-full/permission-denied write surfaces as a clean error

`DiskFaultScenario` seeds a gap entity, revokes write permission on its
parent directory (`0500`, matching `internal/verb/apply_test.go`'s existing
precedent), attempts a real `aiwf promote` against it, and confirms the
refusal is clean: a proper `--format=json` envelope with no panic/stack-trace
markers, no corrupted entity file, no stray temp file, no partial commit.
Only permission-denied is exercised — see Reviewer notes for the
disk-full/ENOSPC scope note. 13 new tests (pure classify table plus
real-binary integration covering every reachable branch, verified non-flaky
over 5 runs); a 6-mutation vacuity probe confirmed every decision branch
actually catches a regression · commits 7194b22a, 0599c2b8 · tests 13/13.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- G-0391 — mutating verbs' lock-busy refusal ignores `--format=json`
  (discovered in M-0242/AC-2; out of this milestone's scope —
  `internal/cli/cliutil` isn't among M-0242's surfaces touched)

## Reviewer notes

- AC-4 exercises only the permission-denied fault, not true disk-full
  (`ENOSPC`). Simulating a real disk-full condition needs privileged
  filesystem-quota setup (a loopback device or size-limited mount) this
  sandboxed environment doesn't have; the AC's own text permits either
  fault, and permission-denied exercises the identical code path in
  `pathutil.AtomicWriteFile` (a failed `os.CreateTemp`/write inside the
  target directory) that `ENOSPC` would also hit.
