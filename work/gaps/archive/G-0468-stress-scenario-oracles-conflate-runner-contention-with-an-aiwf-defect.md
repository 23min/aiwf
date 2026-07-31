---
id: G-0468
title: Stress scenario oracles conflate runner contention with an aiwf defect
status: addressed
priority: high
addressed_by_commit:
    - ac4ce1955
---
## What's missing

Four stress scenarios assert properties of the machine they run on rather than properties of aiwf, so runner contention is reported as an aiwf defect.

- **Deadline oracles.** `classifyConcurrentIDAllocation` and its counterpart in `concurrent_move.go` require all N concurrent actors to succeed *"within repolock's timeout"*. That timeout is a hardcoded two seconds in `internal/cli/cliutil/lock.go`. Under load the tail actors exceed it and receive the documented busy refusal — the verb behaving exactly as specified — which the classifier records as a violation. This is a throughput assertion, not a correctness one.
- **Observation-window oracles.** `mid_write_kill.go` fails with *"never caught the write in flight"* when its poller does not observe the sibling temp file before the target process finishes. Under CPU starvation the atomic-write property still holds; the scenario merely fails to sample it. `lock_kill.go` carries the same shape.

The correctness properties these scenarios exist to test — mutual exclusion, distinct id allocation, no torn writes, a check-clean tree afterwards — are hermetic and hold regardless of timing. They are entangled with the timing assertions inside the same classifier.

## Why it matters

18 of the 100 `go.yml` runs preceding 2026-07-30 failed on these scenarios, spanning 2026-07-11 to 2026-07-29 without interruption. That is the chronic component of the red gate G-0457 records; the `govulncheck` component it names alongside was an eleven-day burst that has since closed.

The flake tracks co-tenancy, not machine size. Run in isolation on four cores, both stress packages pass five repeats out of five, at roughly 38 seconds per run of `internal/stresstest`. Run co-tenant with the full `go test ./...` on those same four cores, the package takes 66.7 seconds — matching the 65 to 77 seconds observed in CI. The scenarios are sound when they own the machine and unsound when they share it, which is a statement about the oracle, not about the runner.

This blocks G-0400. Widening the catalog from 10 verbs to 38 multiplies the flake surface for as long as the oracles conflate contention with defect. It also constrains what G-0457's tourniquet could achieve. That patch splits the scenarios by oracle shape — hermetic ones stay on the every-push path, the ones that race real processes or wait on an observation window move behind a `stress` build tag — which leaves `internal/stresstest` at 62.8% statement coverage in the default lane, down from 85.5%. Every scenario driver G-0400 adds or touches in the tagged class therefore faces the diff-scoped coverage gate with no covering test in that lane. Hermetic oracles are what turn "may these run in the default lane" back into a cost decision rather than a correctness one.

G-0438 records the same defect class one lane over: `flake-hunt.yml` fails on the same runner for the same reason, naming these same packages. Any destination that shares a runner with a broad test sweep reproduces this.

## Resolution shape

Separate the hermetic assertion from the timing assertion in each classifier. The correctness oracle asserts that every successful actor allocated a distinct id, that no actor failed for a reason other than a recognized busy refusal, that at least one actor succeeded so a genuine deadlock is still caught, and that the resulting tree is check-clean beyond the scenario's declared baseline.

The throughput assertion — all N succeed within the lock timeout — is dropped rather than relocated to the on-demand lane. It measures the machine in every lane, not only under CI contention, so a controlled lane would preserve an unsound signal rather than rescue a sound one; and `make stress` is on-demand and never scheduled, so nothing watches what it would preserve. Total deadlock, the property worth keeping from it, is covered by requiring at least one actor to succeed. Pinning lock throughput, should that ever be wanted, belongs in a purpose-built benchmark rather than in a correctness classifier.

Dropping it leaves one property unpinned on the every-push path: that a mutating verb *waits* for a contended lock rather than refusing at once. Setting `AcquireRepoLock`'s timeout to zero passes every test CI runs, and is caught only by `make stress-tests`, which is on-demand. The property belongs to the lock's own contract rather than to a contention classifier, so it is pinned separately — one contender taking the lock after its holder releases it, asserting the contender waited and succeeded, which claims nothing about how many actors get through or how fast.

That pin takes two arms rather than one, because the sharp observation and the realistic one cannot be the same test. The window a wait-assertion depends on is the gap between launching the contender and its lock attempt: on the shared helper that gap is a single function call, while through a real verb it is a whole cobra dispatch, which under CPU starvation stretches far enough that the test passes without exercising the property. So the helper carries the sensitive pin and the verb carries the seam — that a real verb blocks and resumes on the helper's terms, which is where a verb acquiring on its own would surface.

For the observation-window scenarios, the clock goes away rather than being retried against or reclassified. Each observation ends when the watched process acts or exits, whichever comes first, so a slow machine delays the verdict instead of falsifying it. What is left on a clock is a backstop for a process that neither acts nor exits, set far above any plausible slowness — a holder that dies is already caught the moment its output closes, so only a genuine wedge reaches it.

That leaves the sampling miss: a promote that completes before the poller sees its temp file. Missing says nothing about aiwf, so it is retried from the same starting state — which requires rewinding the repo, since the unsampled promote committed its own change and the next attempt would otherwise be a same-state NoOp that writes nothing. Only a run where every attempt misses is reported. Reporting the miss as a third outcome distinct from a violation was the alternative, and it is the wrong default here: it would leave the property unchecked precisely on the loaded runs, and silently.

## Where to fix

- `internal/stresstest/concurrent_id_allocation.go`, `internal/stresstest/concurrent_move.go` — the deadline arm of each classifier.
- `internal/stresstest/concurrent_milestone_race.go` — the same conflation in a scenario that stays tagged for cost, in both its refusal-code arms and in a bound requiring the AC's transition to land exactly once, which contention can prevent.
- `internal/stresstest/mid_write_kill.go`, `internal/stresstest/lock_kill.go` — the observation-window failures.
- `internal/cli/cliutil/lock_test.go` — where the lock-wait property itself is pinned, since no contention classifier should carry it.
- `internal/cli/integration/lock_test.go` — the seam arm of the same property, and the ceiling on the wait its existing busy-refusal test already bounds.
- The paired `*_classify_test.go` files, which pin each classifier against fabricated outcomes and are where the revised oracle is specified. Each of the four scenarios above already has one in the untagged lane, so the revised oracle has a home without new scaffolding.

## Sequencing

Four changes, each landing as its own patch:

1. **Deadline oracles** — `concurrent_id_allocation.go` and `concurrent_move.go`. The replacement oracle is specified above; little is left to decide. Their driver tests return to the untagged lane once the oracle is hermetic, which costs the every-push path a few seconds — the whole untagged package runs in about 16 seconds under `-race` on four cores.
2. **Restoring the lock-wait pin** — a `cliutil` arm and an `internal/cli/integration` arm asserting a contender waits for a released lock rather than refusing at once, split as described above. Independent of the others, and it closes the hole the first change opens.
3. **Observation-window oracles** — `mid_write_kill.go` and `lock_kill.go`, bounded by the watched process as described above.
4. **Recovering the stranded decision tests** — the file splits described below.

The order is load-bearing at one point only: the layout chosen in the last change should be settled against the revised oracles rather than fixed first and then disturbed. The others are independent of each other and share a motivation rather than a line of code.

`concurrent_milestone_race.go` carries the same conflation but stays tagged for its own reasons, so making its oracle hermetic rides along with the last change rather than being a fifth patch of its own. Its refusal-code arms excuse the busy refusal, and its promote-commit bound drops from exactly-one to at-most-one, since contention can legitimately leave no promote landing.

Relaxing that bound needs care, because it was carrying a second property besides mutual exclusion: that a promote reporting success actually did something. A floor requiring some actor through does not recover it — a promote that converges to a NoOp reports `ok` — so the two are separated. The floor catches a race where nothing got through; a distinct arm catches a promote that reported `ok` while no `open -> met` commit exists anywhere, which is a lost mutation rather than a slow machine. Its driver test needs the same treatment: what it asserts about the AC's final status and the commit count becomes conditional on some promote having reported `ok`, rather than assumed.

## Stranded hermetic unit tests

Four pure-decision tests sat inside `stress`-tagged driver files and left the every-push lane as collateral of G-0457's split, because every tagged file holds at least one real-subprocess driver alongside them. None of them touches a subprocess, a clock, or a goroutine; each is a fabricated-input decision test of exactly the kind the untagged lane is for. Each moves to an untagged sibling named for the subject it covers rather than for the scenario that first needed it:

- `patchExactlyOnce` and its test — `concurrent_milestone_race_patch_test.go`. The helper moves with the test, because it was itself defined in a tagged test file; an untagged file compiles into the `stress` build too, so the tagged regression probe still reaches it.
- `TestReadGapFile_ErrorsWhenNoneOrMultipleMatch` — `gitrepo_test.go`, beside the shared fixture helper it covers.
- `TestCrossWorktreeIDRaceScenario_ReconcileErrorsWhenAnActorDidNotSucceed` — `cross_worktree_id_race_reconcile_test.go`.
- `TestWaitForTempFile_ErrorsOnUnreadableDir` — recovered as `TestTempFilePresent_ErrorsOnUnreadableDir` in `mid_write_kill_observe_test.go`, alongside the observation helpers the third change introduced.

The census is worth re-deriving rather than trusting: a file is a candidate only if no test in it references a real binary, a subprocess, a git invocation, a sleep, or a goroutine, and that is a mechanical check over the tagged files rather than a list to maintain.

That split belongs here rather than in its own change: this gap rewrites what those same classifiers assert, so the file layout is chosen once, against the revised oracles, rather than settled first and then disturbed.
