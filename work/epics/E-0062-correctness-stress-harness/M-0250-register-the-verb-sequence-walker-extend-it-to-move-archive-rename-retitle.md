---
id: M-0250
title: Register the verb-sequence walker; extend it to move/archive/rename/retitle
status: in_progress
parent: E-0062
depends_on:
    - M-0249
tdd: required
acs:
    - id: AC-1
      title: cmd/stresstest registers and can run the verb-sequence walker standalone
      status: met
      tdd_phase: done
    - id: AC-2
      title: the walker's legal-transition set includes move, archive, rename, and retitle
      status: met
      tdd_phase: done
    - id: AC-3
      title: a post-step invariant cross-checks aiwf list's output against ground truth
      status: met
      tdd_phase: done
    - id: AC-4
      title: a dedicated concurrency scenario exercises aiwf move across two epics
      status: open
      tdd_phase: red
---

# M-0250 — Register the verb-sequence walker; extend it to move/archive/rename/retitle

## Goal

Register the existing (but unregistered) verb-sequence random-walk scenario into `cmd/stresstest`'s catalog, extend its legal-transition set to drive `move`, `archive`, `rename`, and `retitle`, add a post-step invariant that cross-checks `aiwf list`'s output against ground truth, and add one dedicated true-concurrency scenario for `move`.

## Context

M-0249 built the scenario registry and wired `--scenario all`/`list` to the 12 real scenarios spanning three mechanisms: true simultaneity (multiple OS processes racing on one repo), divergent worktrees reconciled later, and crash/fault injection. G-0400 quantified the result: those 12 scenarios collectively exercise only 10 of 38 leaf CLI verbs, and 15 of the verbs wired for diagnostic logging have zero scenario coverage of any kind — `move`, `archive`, `rename`, and `retitle` among them.

`internal/stresstest/verb_sequence.go` (built during M-0241, never registered — G-0399) is a fourth mechanism the existing 12 don't use: a single-process FSM random walk driving many sequential `aiwf` invocations through legal transitions, checking invariants after each step. That mechanism is the right fit for `move`/`archive`/`rename`/`retitle`'s actual risk profile — accumulated-history bugs (stale cross-references, desynced slugs, a `list` row that doesn't reflect a prior step), not simultaneity — since a long sequence of legal operations is exactly what a random walk is built to generate, and it already exists.

## Acceptance criteria

### AC-1 — cmd/stresstest registers and can run the verb-sequence walker standalone

`cmd/stresstest/registry.go` gains a `verb-sequence` entry adapting `stresstest.NewVerbSequenceScenario` (or its current constructor) into the registry's `scenarioBuilder` shape, closing G-0399. `cmd/stresstest list` names it; `--scenario verb-sequence` runs it alone; `--scenario all` includes it in the combined report. No change to `NewVerbSequenceScenario`'s own constructor signature (per the registry's own adaptation-happens-at-the-call-site convention, G-0397).

### AC-2 — the walker's legal-transition set includes move, archive, rename, and retitle

The FSM's transition table gains `move`, `archive`, `rename`, and `retitle` as selectable operations alongside whatever it already drives, each reachable with nonzero probability during a walk of realistic length. A test asserts the transition table names all four (a structural assertion against the table, not a probabilistic "did a run happen to pick it" check).

### AC-3 — a post-step invariant cross-checks aiwf list's output against ground truth

After each step of the walk, an invariant assertion runs `aiwf list` against the scenario's repo and compares its row set to an independently-derived ground truth (walking the tree directly via the `tree`/`entity` packages, or cross-referencing `aiwf check`/`show` — not re-deriving the comparison through `list`'s own code path, or the check is vacuous). Any divergence (a row `list` should show but doesn't, or vice versa; a stale status/title/parent field) fails the scenario with enough detail (the step that produced it, the specific field) to reproduce without re-running the whole walk.

### AC-4 — a dedicated concurrency scenario exercises aiwf move across two epics

A new named scenario, structurally mirroring `archive-during-active-scope`'s shape, drives real concurrent `aiwf move` invocations moving entities between two epics under real process load — `move` is the one verb in this set whose cross-entity fan-out (source epic, target epic, the moved entity itself) makes a true race worth checking on top of the sequential walk's coverage. Registered in the catalog alongside `verb-sequence`; named per the existing catalog's naming convention (lowercase-hyphenated).

## Constraints

- The walker's ground-truth check (AC-3) must not call through `list`'s own `BuildListRows`/`BuildListCounts` to derive its expected value — that would validate `list` against itself. Derive expected state from the tree/entity packages directly, or from `check`/`show`'s independent code path.
- No change to `internal/stresstest`'s existing 12 scenario constructors' signatures — extensions to `verb_sequence.go`'s transition table and a new `move` scenario are additive.
- Follows the epic's standing invariant-not-eyeball oracle rule: every new assertion (AC-2's transition-table test, AC-3's list-vs-ground-truth check, AC-4's move-race oracle) is a deterministic pass/fail, not something a human reads off output.

## Design notes

- AC-3's invariant lives inside the walker (a post-step hook), not as a standalone scenario — `list` is read-only and cannot itself corrupt state, so there is nothing for a dedicated scenario to race; the risk is entirely in whether `list`'s view drifts from ground truth after other verbs mutate the tree.
- AC-4 is deliberately a separate scenario, not a fifth transition added to the walker — true concurrency (multiple simultaneous processes) and single-process sequential random-walking are different mechanisms with different oracles, and mixing them in one scenario would blur which mechanism caught a given violation.

## Surfaces touched (optional)

- `cmd/stresstest/registry.go`, `cmd/stresstest/list.go`
- `internal/stresstest/verb_sequence.go`
- new `internal/stresstest/move_during_active_scope.go` (or similar; exact name decided during implementation)

## Out of scope

- A dedicated concurrency scenario for `archive`, `rename`, or `retitle` — the sequential walk (AC-2) is judged sufficient for those three; only `move`'s cross-entity fan-out earns a true-race scenario (AC-4).
- Extending scenario coverage to `import`, any `contract` sub-verb, or `worktree add` — G-0400 flags these as open questions for a future milestone, not this one.
- Wiring `list`'s own diagnostic-logging (already done, unrelated to this milestone's scenario-coverage focus).

## Dependencies

- M-0249 — built the scenario registry this milestone extends.
- G-0399 — the gap this milestone's AC-1 closes.
- G-0400 — the gap whose findings motivated this milestone's scope.

## References

- G-0399 — VerbSequenceScenario isn't registered in cmd/stresstest's catalog
- G-0400 — Stress scenario catalog exercises only 10 of 38 aiwf verbs

---

## Work log

### AC-1 — cmd/stresstest registers and can run the verb-sequence walker standalone

Registered, closing G-0399 · commit 59f00c89 · tests: cmd/stresstest package green (race mode).

### AC-2 — the walker's legal-transition set includes move, archive, rename, and retitle

Weighted operation table + move/rename/retitle/archive step methods; fixed a real G-0398 edge case (isEpicAlreadyArchivedRefusal); filed G-0401 · commit fcb2b45a · tests: internal/stresstest + cmd/stresstest packages green (race mode), manual branch-coverage + vacuity audit done.

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- G-0401 — AC-2's coverage audit found that the walker's epic-then-
  milestone creation order means the epic very often reaches a
  terminal status (via its own random promote draws) before the
  milestone is created, so `move`'s practical exercise inside the
  sequential walker is rarer in real usage than its selection weight
  suggests. AC-2's own literal bar (`move` structurally present with
  nonzero weight, and reachable via a dedicated unit test) is met;
  improving the walker's actual hit rate is deferred to G-0401.

## Reviewer notes

- (none)
