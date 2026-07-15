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
      status: met
      tdd_phase: done
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

### AC-2 — Oracle distinguishes a legitimate race from a guard violation

`classifyMilestoneRaceOutcomes` judges each race's outcomes on two
independent signals: outcome-shape/refusal-reason (exactly one promote and
zero-or-one cancel actor land `ok`, every other actor refused with the
matching typed FSM/guard code), and commit-order causality (a winning
cancel's commit, read back via `aiwf-verb`/`aiwf-entity` trailers, must land
strictly after the AC's own `open -> met` commit — the signal that actually
catches the G-0335 shape, since final state alone can't distinguish it from
a legitimate race). Landed alongside a narrow `internal/verb` fix: AC
status/tdd_phase transition refusals and `Cancel`'s already-terminal refusal
carried bare, untyped errors, exiting `2` (usage) instead of `1` (findings)
— the oracle depends on the typed code to tell a legitimate refusal from an
unexpected one · commit 4215e0e0 · tests 15/15.

### AC-3 — Re-running against a reintroduced G-0335-shaped regression fails the run

Builds a disposable, isolated `git worktree` copy of this module with BOTH
the open-AC cancel guard and its `milestone-cancelled-incomplete-acs`
check-rule backstop removed, then runs `ConcurrentMilestoneRaceScenario` 30
times against a binary built from that copy — isolating that
`classifyMilestoneRaceOutcomes`'s own oracle, not the pre-existing
check-rule, provides the protection. Detects the regression in ~19/30
attempts (0/30 false positives against the unpatched binary), mirroring
G-0410's own repeat-N-times empirical methodology. Never touches this
worktree's own tracked source — the patched copy and its worktree
registration are torn down in `t.Cleanup` · commit 640801c5 · tests 4/4.

## Decisions made during implementation

- **`internal/verb` refusal-code fix landed alongside AC-2, not as a separate
  patch.** `promoteAC`/`PromoteACPhase`'s illegal-transition refusal and
  `Cancel`'s already-terminal refusal carried bare, untyped errors (exit `2`
  usage) instead of the typed `CodeFSMTransitionIllegal` every other legality
  refusal in the codebase carries (exit `1` findings) — AC-2's oracle depends
  on the typed code to distinguish a legitimate refusal from an unexpected
  one, so a weaker oracle tolerating untyped codes would have silently
  defeated its own purpose. Verified via a repo-wide caller audit (every
  non-test and production call site of `Cancel`/`PromoteACPhase`/`Promote`)
  that nothing depends on the old bare-error shape or exit code; message text
  unchanged. Not filed as a separate ADR/D-NNN — a narrow, mechanically
  necessary consistency fix with its rationale already captured in the
  commit message and the AC-2 entry above, not an architectural choice.
- **Stale test name fixed in-context at wrap review.**
  `cmd/stresstest/registry_test.go`'s `TestScenarioNames_ListsAllTwelveInCatalogOrder`
  already guarded 15 entries before this milestone (its assertion was
  length-based, not hardcoded — only the name misled); this milestone's
  16th catalog entry widened the gap. Renamed to
  `TestScenarioNames_ListsEveryCatalogEntryInOrder` rather than deferred,
  since this milestone already touched the file.

## Validation

- `go build ./...` — clean.
- `go test ./... -race -parallel 8 -count=1` — 69/69 testable packages `ok`, zero failures.
- `golangci-lint run ./...` — 0 issues.
- `aiwf check` — 0 errors, 6 warnings, all pre-existing and unrelated to this
  milestone (archive-sweep backlog on G-0401, `epic-active-no-drafted-milestones`
  now that M-0257/M-0258 are both terminal, `provenance-untrailered-scope-undefined`).
- Independent two-lens pre-wrap review: code-quality (`wf-review-code`) —
  **approve**, no blocking findings, own mutation probes against
  `classifyMilestoneRaceOutcomes` all caught; design-quality (`wf-rethink`,
  blind reconstruction) — **keep**, no rewrite warranted.
- AC-3's regression detection rate measured independently three times
  (implementer, orchestrator, reviewer): 19/30, 19/30, 13/30 — consistently
  well above the AC's own ≥1/30 floor.

## Deferrals

- (none)

## Reviewer notes

- **Commit-order causality, not final state, is what AC-2's oracle actually
  depends on.** A cancel actor winning the race is only classified as a
  violation if its commit is not strictly after the AC's own `open -> met`
  commit (read via `aiwf-verb`/`aiwf-entity` git trailers) — final state
  alone (AC `met`, milestone `cancelled`) is identical whether the guard
  held or not, which is exactly why AC-1's own end-state assertions
  couldn't have caught the G-0335 shape and AC-2's oracle was necessary.
- **AC-3's detection rate is inherently probabilistic (~13-19/30), not a
  design gap.** The regression only surfaces when real OS scheduling lands
  a cancel actor's commit before the promote actor's — when promote happens
  to win the lock race first even against the regressed binary, the outcome
  is genuinely legitimate (the AC really was `met` first), so 30/30 was
  never the expected signal. The code-quality review's own independent run
  (13/30) and the milestone's original measurement (19/30) bracket the same
  underlying rate; the ≥1/30 assertion plus the reported hit rate together
  are the honest evidence, not an inflated single number.
- **A theoretical, practically-unreachable oracle gap, flagged by the
  code-quality review**: the oracle treats any refusal code other than the
  expected FSM/guard codes as a violation, which technically includes an
  uncoded repo-lock-busy refusal. The lock's retry window (2s) comfortably
  exceeds 8 actors' real contention window and this has never fired across
  every run in this milestone's history (implementer's, reviewer's, and
  this wrap's own). If it ever did, it would be a spurious flake in the
  oracle's favor (an over-eager violation on healthy contention), never a
  missed real one — no action taken, tracked here rather than as a gap
  since it's a documented, shared, negligible-risk property of the harness's
  repolock-serialization design, not deferred work.
- **The exit-code change (usage→findings on three refusal paths) is
  user-visible** and may warrant a `CHANGELOG.md` `[Unreleased]` line at the
  epic's release — not done here per this repo's own release-process
  convention (CHANGELOG updates land at tag time), but flagged for
  `aiwfx-release`/`aiwfx-wrap-epic` to pick up.
