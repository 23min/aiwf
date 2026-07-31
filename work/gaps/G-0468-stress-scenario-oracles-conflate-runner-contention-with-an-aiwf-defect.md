---
id: G-0468
title: Stress scenario oracles conflate runner contention with an aiwf defect
status: open
priority: high
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

For the observation-window scenarios, either retry the sampling loop until it observes the window, or report a failure to sample as an outcome distinct from a violation.

## Where to fix

- `internal/stresstest/concurrent_id_allocation.go`, `internal/stresstest/concurrent_move.go` — the deadline arm of each classifier.
- `internal/stresstest/mid_write_kill.go`, `internal/stresstest/lock_kill.go` — the observation-window failures.
- The paired `*_classify_test.go` files, which pin each classifier against fabricated outcomes and are where the revised oracle is specified. Each of the four scenarios above already has one in the untagged lane, so the revised oracle has a home without new scaffolding.

## Sequencing

Three changes, each landing as its own patch:

1. **Deadline oracles** — `concurrent_id_allocation.go` and `concurrent_move.go`. The replacement oracle is specified above; little is left to decide.
2. **Observation-window oracles** — `mid_write_kill.go` and `lock_kill.go`. Carries the one open decision: retry the sampling loop until it observes the window, or report failure-to-sample as an outcome distinct from a violation.
3. **Recovering the stranded decision tests** — the file splits described below.

The order is load-bearing at one point only: the layout chosen in the third change should be settled against the revised oracles rather than fixed first and then disturbed. The first two are independent of each other and share a motivation rather than a line of code.

## Stranded hermetic unit tests

Four pure-decision tests sit inside `stress`-tagged driver files and left the every-push lane as collateral of G-0457's split, because every tagged file holds at least one real-subprocess driver alongside them:

- `TestPatchExactlyOnce` — `concurrent_milestone_race_regression_test.go`
- `TestReadGapFile_ErrorsWhenNoneOrMultipleMatch`, `TestWaitForTempFile_ErrorsOnUnreadableDir` — `mid_write_kill_test.go`
- `TestCrossWorktreeIDRaceScenario_ReconcileErrorsWhenAnActorDidNotSucceed` — `cross_worktree_id_race_test.go`

None of them touches a subprocess, a clock, or a goroutine; each is a fabricated-input decision test of exactly the kind the untagged lane is for. Recovering them means splitting each driver file so the decision tests live in an untagged sibling.

That split belongs here rather than in its own change: this gap rewrites what those same classifiers assert, so the file layout should be chosen once, against the revised oracles, rather than settled first and then disturbed.
