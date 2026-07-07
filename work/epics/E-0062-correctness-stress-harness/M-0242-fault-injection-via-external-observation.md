---
id: M-0242
title: Fault injection via external observation
status: draft
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: A process killed while holding repolock releases it via kernel fd cleanup
      status: open
      tdd_phase: red
    - id: AC-2
      title: A process killed mid-write never leaves a half-written entity file
      status: open
      tdd_phase: red
    - id: AC-3
      title: Lock-held and temp-file states are detected with no production-code change
      status: open
      tdd_phase: red
    - id: AC-4
      title: A disk-full or permission-denied write surfaces as a clean error
      status: open
      tdd_phase: red
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

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
