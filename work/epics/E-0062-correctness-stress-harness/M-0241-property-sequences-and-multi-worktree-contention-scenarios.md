---
id: M-0241
title: Property sequences and multi-worktree contention scenarios
status: in_progress
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: Generated verb sequences are checked for FSM legality each step
      status: met
      tdd_phase: done
    - id: AC-2
      title: Real subprocesses racing repolock never produce a duplicate id
      status: met
      tdd_phase: done
    - id: AC-3
      title: A cross-worktree id race is always caught and resolved by reallocate
      status: open
      tdd_phase: red
    - id: AC-4
      title: repolock's per-worktree lockfile scoping is confirmed intentional
      status: open
      tdd_phase: red
    - id: AC-5
      title: A sibling worktree's commit is confirmed unreachable from another's check
      status: open
      tdd_phase: red
---

## Goal

Implement scenario tiers 1 (single-process verb-sequence properties) and 2
(multi-process/multi-worktree contention) against the M-0240 skeleton, and
empirically confirm two open architectural questions rather than assume
them.

## Context

M-0240 shipped the driver, scenario interface, and streaming report with
only a placeholder scenario. This milestone is the first to exercise real
aiwf behavior through it — both the "loose/statistical race" mechanism
(start actors close together, repeat) that the initiative doc's
Orchestration mechanics section describes as the default for this tier.

## Acceptance criteria

### AC-1 — Generated verb sequences are checked for FSM legality each step

Extends the existing `internal/entity/transition_property_test.go` pattern:
random (legal and illegal) verb sequences run against one real temp repo
via the real binary; after every step, `aiwf check` is clean or
expected-findings-only, and exactly one commit landed per mutation.

### AC-2 — Real subprocesses racing repolock never produce a duplicate id

N real `aiwf` subprocesses, started close together against one working
copy, contend for an id allocation. Repeated per M-0240's `--repeat`
mechanism. No run ever produces two entities with the same id.

### AC-3 — A cross-worktree id race is always caught and resolved by reallocate

The same contention, but across sibling worktrees of one repo rather than
one working copy. A duplicate id here is an *accepted* outcome (per this
repo's own eventual-consistency design for id allocation) — the assertion
is that `aiwf check` always surfaces it as a finding, and `aiwf reallocate`
always resolves it cleanly, never that the duplicate never happens.

### AC-4 — repolock's per-worktree lockfile scoping is confirmed intentional

`repolock.lockfilePath` resolves differently for a linked worktree (whose
`.git` is a file, not a directory) than for the main checkout — meaning a
worktree does not share the main checkout's lock. This scenario exercises
that directly and confirms the consequence is exactly AC-3's
accepted-eventual duplicate-id-then-reallocate story, not some other,
unaccounted-for failure mode.

### AC-5 — A sibling worktree's commit is confirmed unreachable from another's check

Confirms the documented architectural fact surfaced while scoping this
epic (`branch_scenarios_ac2_test.go`'s comment on `aiwf check`'s provenance
walk being `git log HEAD`, not `--all`): a commit made in a sibling
worktree is invisible to a check run in another worktree. This scenario
checks whether that invisibility has any *other* consequence beyond the
one rule (isolation-escape) that already documents it — if a different
rule implicitly assumes broader reachability, this is where that surfaces.

## Constraints

- Tier 2 scenarios use the loose/statistical race mechanism only — start
  actors close together, repeat; no directed/failpoint mechanism in this
  milestone (that's reserved, per the epic's constraints, for a case tier 3
  or 4 prove genuinely needs it).
- Every scenario's pass/fail oracle is the invariant itself (id uniqueness,
  FSM legality, reachability), never a human reading harness output.

## Design notes

- AC-3 and AC-4 are deliberately framed as *confirming* an accepted design,
  not fixing a bug — don't treat a duplicate id under AC-3 as a failure;
  treat an *unresolved* or *undetected* duplicate id as one.

## Surfaces touched

- `internal/stresstest/` (new scenario files)
- `internal/repolock/` (read-only — no production changes expected from
  this milestone; AC-4/AC-5 are read-only confirmations)

## Out of scope

- Fault injection (kill -9, disk-full) — M-0242.
- The named G-0212/G-0269 scenarios — M-0243.
- Any change to `repolock` or the provenance-walk's reachability behavior —
  this milestone confirms current behavior, it doesn't change it. A change,
  if one turns out to be warranted, is a new gap, not this milestone's job.

## Dependencies

- M-0240 — the harness skeleton.

## References

- `docs/initiatives/robustness-correctness-stress-testing.md`
- `internal/cli/integration/branch_scenarios_ac2_test.go` (the
  `git log HEAD`-not-`--all` reachability documentation)

---

## Work log

### AC-1 — Generated verb sequences are checked for FSM legality each step

`VerbSequenceScenario` walks random legal/illegal `aiwf promote` attempts against one entity of every kind, via the real compiled binary, in one disposable repo — extending `internal/entity/transition_property_test.go`'s FSM-property pattern to the real binary. Discovered and handled a real nuance: an FSM-legal transition can still be refused by an orthogonal business rule (gap's `addressed`-needs-`--by` resolver gate) distinct from FSM illegality; the classifier treats that as a legitimate refusal, not a violation. · commit 0320f740 · tests 25/25

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
