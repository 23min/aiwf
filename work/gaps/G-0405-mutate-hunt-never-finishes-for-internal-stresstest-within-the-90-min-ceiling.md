---
id: G-0405
title: mutate-hunt never finishes for internal/stresstest within the 90-min ceiling
status: open
---
## What's missing

A real, scoped `mutate-hunt` dispatch against `./internal/stresstest`
(corrected package pattern, see G-0403) ran for the full 90-minute job
timeout and was force-canceled by GitHub Actions without finishing —
confirmed via `gh run view --log` (run 29148292046): the log shows
individual per-mutant `KILLED`/`LIVED`/`NOT COVERED`/`TIMED OUT`
results streaming steadily, then `##[error]The operation was
canceled.` at the 90-minute mark, having reached only as far as
`verb_sequence.go:439` out of the whole package. No `mutate-report.json`
was ever written (the "Upload report" step found nothing to upload),
so the run produced zero usable machine-readable output despite
finding many real mutants along the way.

Per-mutant timing in the log runs ~10-20s typically, with at least one
`TIMED OUT` outlier taking ~6 minutes for a single mutant
(`INCREMENT_DECREMENT at verb_sequence.go:195:28`, 11:05:47 →
11:11:52). At `--workers 1` (required per the workflow's own doc
comment, to avoid test-binary build-cache contention) this package
alone — with its dozens of files, each carrying real logic mutable by
gremlins' 5 operators — plausibly has several hundred mutants total.

## Why it matters

`internal/stresstest`'s own tests are unusually expensive per
invocation compared to the rest of the repo: most tests that exercise
a real scenario call `sharedTestBinary(t)`, which builds the real
`aiwf` binary via a `sync.Once`-shared fixture — but gremlins launches
a **fresh `go test` process per mutant**, and `sync.Once` state doesn't
survive across process boundaries, so every single mutant re-pays the
full `aiwf` build cost on top of the mutated code's own test run. The
`--workers 1`/`--timeout-coefficient 15` tuning in the workflow's own
header comment was set for "this codebase" generically; it doesn't
account for this package's structurally different (much higher)
per-test-invocation cost.

Net effect: mutation testing `internal/stresstest` via the current
workflow, as configured, cannot complete — not "runs slow," but
"never produces a report." A scoped dispatch aimed at exactly the
package this session most wanted mechanically-confirmed test-quality
signal for is the one package the workflow can't actually finish for.

## Direction

Worth a real decision, not assumed. Candidate directions:

- Scope narrower still — dispatch against individual files or a
  sub-selection within `internal/stresstest` across multiple runs,
  rather than the whole package at once.
- Raise the job's `timeout-minutes` specifically when `pkg_pattern`
  targets this package (or make it operator-configurable per
  dispatch, not hardcoded at 90).
- Exclude the package's slowest real-subprocess tests from the
  coverage-gathering pass gremlins uses (e.g., via `-short`-style
  scoping) — but this cuts against the point of mutation-testing the
  scenario logic itself, which is exactly the code that calls
  `sharedTestBinary`.
- Accept that this package needs a dedicated, longer-running,
  possibly `--workers`-tuned-differently invocation outside the
  general-purpose `mutate-hunt` workflow, and document that
  expectation explicitly.

## Scope

`.github/workflows/mutate-hunt.yml` (timeout/worker tuning, possibly
per-pattern), or operator documentation about expected runtime for
this specific package (ties into G-0404's own stress-harness
documentation gap).