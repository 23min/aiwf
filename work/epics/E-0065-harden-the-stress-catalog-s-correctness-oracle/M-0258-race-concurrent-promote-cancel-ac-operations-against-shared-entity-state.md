---
id: M-0258
title: Race concurrent promote/cancel/AC operations against shared entity state
status: in_progress
parent: E-0065
depends_on:
    - M-0257
tdd: required
acs:
    - id: AC-1
      title: N concurrent actors race promote/cancel on one shared milestone+AC
      status: met
      tdd_phase: done
    - id: AC-2
      title: Oracle distinguishes a legitimate race from a guard violation
      status: met
      tdd_phase: done
    - id: AC-3
      title: Re-running against a reintroduced G-0335-shaped regression fails the run
      status: open
      tdd_phase: red
---

## Goal

Add a stress scenario that races N concurrent `aiwf` actors against
promote/cancel/AC operations on *shared* entity state, with an oracle that
distinguishes a legitimate race outcome from an actual guard violation —
closing, for the concurrent-interleaving case, the class of bug G-0335
demonstrated (a verb-time guard missing entirely, with no check-rule
backstop either) that `VerbSequenceScenario`'s sequential walk alone cannot
catch.

## Context

E-0062's `concurrent-writer-at-scale`
(`internal/stresstest/concurrent_writer_at_scale.go`) already proves the
goroutine + `sync.WaitGroup` subprocess fan-out harness shape, racing
concurrent `aiwf cancel` calls — but each actor targets its own distinct
pre-seeded entity, not a shared one, so it validates ADR-0017's `O_APPEND`
diagnostic-log-write safety, not entity-state race safety. G-0410 confirmed
empirically (30/30 clean `verb-sequence` runs against a pre-fix G-0335
binary) that the catalog cannot detect a verb-time guard missing entirely
when walked sequentially in one process against one disposable repo; no
scenario today contests a single shared entity or AC via concurrent
promote/cancel, so that class of bug under real concurrent interleaving is
untested territory regardless of oracle breadth. This milestone is sequenced
after M-0257 (which de-risks the harness's oracle-broadening work first) but
does not depend on any of M-0257's code.

## Acceptance criteria

### AC-1 — N concurrent actors race promote/cancel on one shared milestone+AC

A new scenario (or a concurrent mode added to `VerbSequenceScenario`)
launches N `aiwf` subprocess actors concurrently against ONE shared
disposable repo, reusing `concurrent-writer-at-scale`'s goroutine +
`sync.WaitGroup` fan-out pattern. Unlike that scenario, every actor here
targets the *same* milestone (which carries at least one open AC) with
promote and cancel operations — the exact shape G-0335 exercised (the
open-AC guard on milestone cancel).

### AC-2 — Oracle distinguishes a legitimate race from a guard violation

The scenario's oracle classifies each concurrent outcome set as one of:
**legitimate race** — exactly one actor's operation lands as an FSM-legal
commit and every other concurrently-dispatched actor targeting the same
transition observes a clean refusal with zero commits (or, if it observed
the post-mutation state, a now-different legal transition); or
**violation** — two actors both land a commit for what should have been a
mutually-exclusive transition, or a refusal whose reason contradicts the
FSM's own verdict (`entity.ValidateTransition`) or the domain-specific guard
under test (the open-AC cancel guard). A legitimate race must never be
flagged as a violation — over-eager classification would make every green
run meaningless.

### AC-3 — Re-running against a reintroduced G-0335-shaped regression fails the run

With the milestone-cancel open-AC guard deliberately removed from the
`cancel` verb path, and its check-rule backstop
(`milestone-cancelled-incomplete-acs`) also stubbed out, this scenario's run
fails — reporting at least one violation. Validated via the same
repeat-N-times empirical methodology G-0410 used against the pre-fix G-0335
binary (30 runs), not a single run, since the scenario's outcome depends on
real goroutine/subprocess timing.

## Constraints

- Reuse `concurrent-writer-at-scale`'s existing goroutine + `sync.WaitGroup`
  subprocess fan-out harness; do not build new concurrency machinery from
  scratch.
- The shared milestone's initial AC set must guarantee the open-AC cancel
  guard is live for the whole race window (at least one AC stays `open` for
  the scenario's duration), or AC-3's regression check has nothing to catch.

## Design notes

- Target one shared entity+AC pair per race *round*, not one shared entity
  for the scenario's entire run — each round seeds a fresh milestone+AC pair,
  so the scenario can repeat multiple independent trials within one process
  the same way other scenarios accumulate statistical confidence across a
  `--repeat N` invocation.
- The concurrent actors' operation set is deliberately narrower than
  `VerbSequenceScenario`'s full weighted table (`baseWalkOperations`) —
  scoped to promote and cancel against the shared milestone, since that's
  the exact class of guard G-0410 named, not a general-purpose concurrent
  walker.

## Surfaces touched

- `internal/stresstest/concurrent_writer_at_scale.go` — fan-out harness
  pattern reused
- A new `internal/stresstest/*.go` scenario file (or a concurrent-mode
  extension to `verb_sequence.go`), registered in `cmd/stresstest/registry.go`

## Out of scope

- Racing rename/retitle/archive/move concurrently — `VerbSequenceScenario`'s
  sequential walk already exercises those; this milestone is scoped to the
  promote/cancel/AC guard class G-0410 named.
- Applying M-0257's broadened check-clean oracle to this new scenario's own
  end-state assertion — worth reusing if it fits cleanly, but not this
  milestone's primary deliverable.

## Dependencies

- M-0257 — sequenced after it per epic planning (de-risks the harness before
  adding concurrency to it); not a hard blocking dependency, since this
  milestone doesn't consume any of M-0257's code.

## References

- G-0410 — stress catalog can't detect a missing domain-specific promote guard
- G-0335 — the concrete regression (open-AC cancel guard bypass) this
  scenario reproduces under concurrency
- E-0065 — Harden the stress catalog's correctness oracle (parent epic)

## Work log

### AC-1 — N concurrent actors race promote/cancel on one shared milestone+AC

`ConcurrentMilestoneRaceScenario` (registered as `concurrent-milestone-race`)
races 8 actors — split between `aiwf promote <M>/AC-1 met` and `aiwf cancel
<M>` — against one shared, pre-seeded milestone+AC pair via goroutine +
`sync.WaitGroup` subprocess fan-out. Scoped to AC-1's mechanical invariants
only: every actor returns a parseable envelope, and the resulting tree stays
check-clean beyond a curated baseline. The legitimate-race-vs-guard-violation
oracle is AC-2's own follow-on cycle, built on top of the `raceActorOutcome`
shape this AC already captures · commit 632debc8 · tests 8/8.
