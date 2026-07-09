---
id: M-0244
title: Concurrent-writer test at scale; triage process; epic close
status: in_progress
parent: E-0062
depends_on:
    - M-0241
    - M-0242
    - M-0243
tdd: required
acs:
    - id: AC-1
      title: Concurrent subprocesses sharing one log file never tear or interleave a line
      status: met
      tdd_phase: done
    - id: AC-2
      title: A documented triage procedure turns a found violation into a gap and test
      status: open
      tdd_phase: red
    - id: AC-3
      title: Every success criterion in E-0062's epic spec has a passing demonstration
      status: open
      tdd_phase: red
---

## Goal

Prove E-0061's `O_APPEND` diagnostic-log safety under real multi-process
load (not just the package-level test built in M-0237), document the
triage procedure this epic's findings flow through, and verify the epic's
own success criteria end-to-end before it closes.

## Context

This is the epic's capstone: it depends on all three scenario-tier
milestones (M-0241, M-0242, M-0243) being done. Everything needed to run
this tier already exists by the time this milestone starts — the harness,
the real binary, the logger.

## Acceptance criteria

### AC-1 — Concurrent subprocesses sharing one log file never tear or interleave a line

N real `aiwf` subprocesses, each with `AIWF_LOG=debug` pointed at the same
daily log file, run concurrently. Asserts every resulting line parses
cleanly, every `run_id` appears exactly once, and none is interleaved or
truncated — the harness proving out ADR-0017's Decision #5 under load the
package-level test (M-0237) couldn't exercise on its own, since that test
predates any real verb calling the logger.

### AC-2 — A documented triage procedure turns a found violation into a gap and test

A short, concrete procedure (documented in this milestone's spec or a
pointer from it): a violation the harness surfaces gets a new gap
(`aiwf add gap`) referencing the raw-report event and preserved repo state,
and a minimal regression test is promoted into the normal, every-push
suite — not left living only inside the stress harness.

### AC-3 — Every success criterion in E-0062's epic spec has a passing demonstration

Each checkbox in E-0062's *Success criteria* section is walked and
confirmed against the finished harness — not asserted from memory.

## Constraints

- AC-1's test is at real subprocess scale (multiple `aiwf` binary
  invocations), distinct from and in addition to M-0237's package-level
  goroutine/file-handle test — it's proving the same property under a
  higher-fidelity load, not duplicating the earlier test.
- This milestone doesn't introduce new scenario categories — it closes the
  loop on ones already built.

## Design notes

- If AC-3's walk-through finds a success criterion not actually met, that's
  this milestone's problem to resolve (more scenario work, or a scope
  correction to the epic spec with the user's sign-off) — not something to
  gloss over at wrap.

## Surfaces touched

- `internal/stresstest/` (the concurrent-writer-at-scale scenario)
- This epic's spec (`epic.md`) — finalized at wrap per the usual ritual

## Out of scope

- Any new scenario category not already scoped in M-0241–M-0243.
- Making the harness a CI gate — still out of scope for the whole epic, per
  its own spec.

## Dependencies

- M-0241, M-0242, M-0243 — all three scenario tiers must be done.

## References

- `docs/adr/ADR-0017-opt-in-slog-diagnostic-logging-default-off-xdg-state-home-file-route.md`
- `docs/initiatives/robustness-correctness-stress-testing.md`

---

## Work log

### AC-1 — Concurrent subprocesses sharing one log file never tear or interleave a line

Confirmed: n real `aiwf cancel` subprocesses, each pointed at one shared
diagnostic log file via `AIWF_LOG_FILE`, never tear or interleave a line —
the OS-level `O_APPEND` guarantee holds under genuine separate-process
concurrency, not just the package-level goroutine simulation M-0237 already
covers. Every line's `run_id` matches exactly one real invocation's own
`--format=json` correlation id, extending M-0239's single-process
correlation guarantee to concurrent, multi-process load. A vacuity-probe
mutation initially survived (a message-swap in the foreign-run_id branch —
the test asserted a generic phrase, not which id it was attached to);
strengthened the classify tests to name the specific run_id in every
expected violation, then reconfirmed the mutation is caught · commit
7e0b4237 · tests 13/13

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
