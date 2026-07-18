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
intermittently fails with `actor 2: running aiwf cancel G-0003: exit status 2` (a usage
exit from a concurrent `aiwf cancel`), while the same scenario passes reliably locally
(13/13 under `-race -parallel 8`) and in a full local `make ci`. A re-run of the identical
commit turns the job green, confirming the failure is timing-dependent, not a code defect.

## Why it matters

The `test` job is part of the required `go` gate that runs on every push. A concurrency
scenario that reds-out spuriously trains maintainers to reflexively re-run rather than
read the failure — which is exactly how a genuine concurrency regression would slip
through. It has already spuriously failed at least two consecutive epic-wrap pushes. The
fix is to make the scenario deterministic under CI load: identify whether the `exit
status 2` is an expected-under-contention outcome the oracle should accept (a concurrent
`cancel` losing a race and re-reading), or a genuine lost-update the harness should retry
past, and encode that in the scenario's pass/fail oracle rather than assuming every actor
operation succeeds first try. Surfaced during E-0067's epic-wrap CI run.
