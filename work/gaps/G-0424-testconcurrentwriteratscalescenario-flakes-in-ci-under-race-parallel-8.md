---
id: G-0424
title: TestConcurrentWriterAtScaleScenario flakes in CI under -race -parallel 8
status: open
priority: medium
discovered_in: E-0067
---
## What's missing

`TestConcurrentWriterAtScaleScenario_RealBinary_NConcurrentWritersNeverTearOrInterleave`
(`internal/stresstest`, added by E-0065's stress-catalog hardening) is flaky in the CI
`go` workflow's `test` step. Under `-race -parallel 8` on a loaded GitHub runner it
intermittently fails with `actor N: running aiwf cancel <id>: exit status 2` (a usage
exit from a concurrent `aiwf cancel`), while the same scenario passes reliably locally
(13/13 under `-race -parallel 8`) and in a full local `make ci`. A re-run of the identical
commit turns the job green, confirming the failure is timing-dependent, not a code defect.

## Root cause

The `exit status 2` is a **repo-lock acquisition timeout under contention**, not a lost
update, a torn or interleaved diagnostic-log write, or any defect in the read/write path.
The scenario launches n (=12) concurrent real `aiwf cancel` subprocesses against one repo.
Every mutating verb takes the repo-wide `flock` before doing any work, with a 2-second
acquisition timeout (`internal/cli/cliutil/lock.go`). The n cancels serialize through that
lock, and each critical section forks a `git commit` (~55 ms uncontended). Late-queued
actors wait behind all predecessors; when the cumulative wait exceeds the 2-second
timeout, `repolock.Acquire` returns `repolock.ErrBusy`, which `cliutil.AcquireRepoLock`
maps to exit code 2 with the message "another aiwf process is running on this repo; retry
in a moment".

Locally, 12 serialized commits total ~600 ms — well under 2 s, so every actor completes
(13/13). Under CI's `-race -parallel 8` plus every other test package competing for an
oversubscribed runner, each serialized critical section stretches until the tail actor's
queue-wait crosses 2 s, and that actor exits 2.

The harness then aborts the whole run: `launchCancelActor` uses `cmd.Output()`, which
errors on any non-zero exit, discards the (valid) stdout envelope, and returns an actor
error that `Run` treats as fatal. This contradicts the scenario's own documented oracle: a
lock-busy invocation still emits exactly one clean diagnostic line — its `EmitVerbOutcome`
defer is registered before lock acquisition — so the O_APPEND log-write-safety invariant
the scenario exists to test is never actually violated by a busy exit. The harness aborts
on a verb exit code its own comment calls irrelevant to the scenario's claim.

Reproduced end-to-end with the real binary: a `cancel` against a genuinely held lock waits
~2.06 s and exits 2 with the busy message; 80 concurrent cancels (the scenario's exact
mechanism, no artificial lock-holding) produced 44 busy exits, every one the same message.

## Why it matters

The `test` job is part of the required `go` gate that runs on every push. A concurrency
scenario that reds-out spuriously trains maintainers to reflexively re-run rather than
read the failure — which is exactly how a genuine concurrency regression would slip
through. It has already spuriously failed multiple consecutive epic-wrap pushes.

## Proposed fix

Make the actor tolerate the expected-under-contention busy outcome by **retrying the
`cancel` until it completes**, rather than assuming every actor succeeds on the first
attempt — the busy envelope's own "retry in a moment" is the verb's guidance. In
`launchCancelActor`, detect the lock-busy outcome (the exit-2 usage exit carrying the
busy-envelope error) and retry with a short backoff, bounded by an attempt count or
deadline set well above the 2-second lock timeout so the tail actor's eventual success is
deterministic regardless of runner load. Only a non-busy failure, or exhaustion of the
retry budget, is a real actor error. This preserves the scenario's exact-n-clean-lines
oracle (one successful `cancel` per actor) while removing the timing dependence; the
bounded budget keeps a genuine hang failing rather than looping forever.
