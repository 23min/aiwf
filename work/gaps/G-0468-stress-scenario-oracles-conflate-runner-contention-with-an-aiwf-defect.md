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
- **Unrecognized busy refusals.** `concurrent_writer_at_scale.go` models contention correctly — retry past a busy envelope, bounded budget — but recognizes that envelope by substring match, so a busy refusal it fails to recognize aborts the run as a hard error rather than a classified outcome. G-0467 is the enabler for recognizing it structurally.

The correctness properties these scenarios exist to test — mutual exclusion, distinct id allocation, no torn writes, a check-clean tree afterwards — are hermetic and hold regardless of timing. They are entangled with the timing assertions inside the same classifier.

## Why it matters

18 of the 100 `go.yml` runs preceding 2026-07-30 failed on these scenarios, spanning 2026-07-11 to 2026-07-29 without interruption. That is the chronic component of the red gate G-0457 records; the `govulncheck` component it names alongside was an eleven-day burst that has since closed.

The flake tracks co-tenancy, not machine size. Run in isolation on four cores, both stress packages pass five repeats out of five, at roughly 38 seconds per run of `internal/stresstest`. Run co-tenant with the full `go test ./...` on those same four cores, the package takes 66.7 seconds — matching the 65 to 77 seconds observed in CI. The scenarios are sound when they own the machine and unsound when they share it, which is a statement about the oracle, not about the runner.

This blocks G-0400. Widening the catalog from 10 verbs to 38 multiplies the flake surface for as long as the oracles conflate contention with defect. It also constrains G-0457's resolution: gating the real-binary drivers out of the default run drops `internal/stresstest` statement coverage from 85.5% to 34.7%, so every scenario driver G-0400 adds or touches faces the diff-scoped coverage gate with no covering test in the default lane. Hermetic oracles are what turn "may these run in the default lane" back into a cost decision rather than a correctness one.

G-0438 records the same defect class one lane over: `flake-hunt.yml` fails on the same runner for the same reason, naming these same packages. Any destination that shares a runner with a broad test sweep reproduces this.

## Resolution shape

Separate the hermetic assertion from the timing assertion in each classifier. The correctness oracle asserts that every successful actor allocated a distinct id, that no actor failed for a reason other than a recognized busy refusal, that at least one actor succeeded so a genuine deadlock is still caught, and that the resulting tree is check-clean beyond the scenario's declared baseline. The throughput assertion — all N succeed within the lock timeout — belongs to the on-demand `make stress` lane, where the environment is controlled, not to a classifier shared with the every-push path.

For the observation-window scenarios, either retry the sampling loop until it observes the window, or report a failure to sample as an outcome distinct from a violation.

Recognizing a busy refusal structurally requires the machine-readable code G-0467 introduces, so that gap lands first.

## Where to fix

- `internal/stresstest/concurrent_id_allocation.go`, `internal/stresstest/concurrent_move.go` — the deadline arm of each classifier.
- `internal/stresstest/mid_write_kill.go`, `internal/stresstest/lock_kill.go` — the observation-window failures.
- `internal/stresstest/concurrent_writer_at_scale.go` — busy recognition, jointly with G-0467.
- The paired `*_classify_test.go` files, which pin each classifier against fabricated outcomes and are where the revised oracle is specified.
