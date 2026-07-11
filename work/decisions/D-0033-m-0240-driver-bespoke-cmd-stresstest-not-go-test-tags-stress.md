---
id: D-0033
title: 'M-0240 driver: bespoke cmd/stresstest, not go test -tags=stress'
status: proposed
relates_to:
    - E-0062
    - M-0240
---
> **Date:** 2026-07-08 · **Decided by:** human/peter

## Question

M-0240's design notes deferred one concrete choice to the start of
implementation: how the stress harness drives its scenario catalog.
`docs/initiatives/robustness-correctness-stress-testing.md` named two
candidates — reusing `go test -tags=stress -json` (Go's own streaming,
abort-tolerant test-event format, plus `t.Run`/`-run` for free) or a fully
bespoke `cmd/stresstest` binary. The answer wasn't obvious because the
`go test` route looked like the cheaper, more idiomatic default, and the
initiative doc itself leaned that way.

## Decision

The harness driver is a bespoke `cmd/stresstest` binary (Cobra-based,
matching this repo's CLI convention and the `make diag-aiwf` precedent for
dev-only, worktree-scoped tools), not `go test -tags=stress -json`. It
exposes at least a `run` and a `compose` invocation as separate,
independently-runnable steps.

## Reasoning

`go test -tags=stress -json` was rejected for two reasons that outweighed
its lower build cost:

- AC-3 requires composing a report after the harness itself is `kill -9`'d
  mid-run. Composition must be a standalone re-invocation over whatever the
  raw-report file contains — there is no live process left to continue
  after a hard kill. That is a natural fit for a small CLI's separate
  `compose` subcommand, and an awkward one for `go test`, which has no
  built-in notion of "resume reporting in a fresh process after the last
  one was killed."
- AC-2 requires our own raw-report writer with the exact `O_APPEND`/
  one-`Write()`-per-record discipline `internal/logger` already
  established (ADR-0017 Decision #5) — a mechanism distinct from Go's own
  test2json event stream. Choosing the `go test` route would not actually
  let us reuse that stream for the raw report; we would still build a
  parallel writer, which erodes most of the "reuse existing tooling"
  benefit the `go test` route was chosen for in the first place.

Later milestones in this epic (tiers 2-4: multi-process contention, fault
injection, directed races) need precise control over subprocess spawning
and signal delivery to specific actors. A bespoke driver gives that
directly; wrapping the same orchestration inside `go test`'s process and
subtest model would mean fighting the mechanism rather than using it,
which the milestone's own design notes named as the deciding factor.

The cost accepted: a second CLI surface to build and maintain, separate
from `cmd/aiwf`. This is judged worth it given the abort-safety and
signal-control requirements above.

## Consequences (optional)

- `cmd/stresstest` lives entirely under its own tree, never installed
  alongside `cmd/aiwf`, per E-0062's scope constraints.
- The raw-report writer under `internal/stresstest/` reuses
  `internal/logger`'s `O_APPEND` discipline directly rather than adapting
  Go's test2json format.
- `compose` is invoked independently of `run`, satisfying the
  abort-recovery requirement without special-casing a killed process.
