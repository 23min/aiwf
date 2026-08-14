---
id: M-0308
title: Ship the spec-measurement ritual and settle what its record is
status: draft
parent: E-0085
tdd: required
acs:
    - id: AC-1
      title: A completed pass leaves a record naming the entity whose claims it measured
      status: open
    - id: AC-2
      title: The sweep step carries a runnable command that produces a current-trunk checkout
      status: open
    - id: AC-3
      title: The ritual's measurement step precedes its sweep step
      status: open
---

## Goal

Ship the ritual that measures an entity's factual claims before implementation,
with its terminating record settled, so a later seam has something to invoke and
a later check has something to read.

## Context

No surface in this repo supplies a method for deciding whether a specification is
right. `aiwfx-start-milestone`'s preflight and `wf-patch` step 1 both ask for the
judgment and stop there; this milestone builds the method they will call, and
M-0309 wires them.

The three parts and their ordering are settled by E-0085 and are not re-opened
here. What is open is the record: the epic names it the one blocking question,
because a record whose shape nobody fixed is a record no later rule can read.

## Acceptance criteria

### AC-1 — A completed pass leaves a record naming the entity whose claims it measured

Running the ritual to completion produces one of exactly two outcomes, and the
ritual names both: an `aiwf edit-body <id>` commit against the entity whose
claims were measured, or a recorded no-change. A pass that measured nothing and
a pass that was never run are distinguishable afterwards by reading history.

The record's shape is settled by this criterion, not assumed — commit trailer,
body line, or empty `edit-body` are the candidates, and the one chosen is
whatever a later rule could read without parsing prose. Capture the choice via
`aiwfx-record-decision`; a shape with no recorded reasoning is a shape the next
reader re-litigates.

### AC-2 — The sweep step carries a runnable command that produces a current-trunk checkout

The sweep reads current trunk, and the ritual gets the operator there by naming
a command rather than by asking them to remember. Run against a stale branch the
sweep missed the most consequential finding of its motivating episode and
produced one confident false one, so this is the step's load-bearing
precondition rather than a nicety.

The assertion derives both sides: the command the sweep names resolves against
the surface that defines it, so a rename breaks the test rather than leaving the
prose pointing at nothing. An assertion that the section merely *contains* a
string proves only that the string is present and is the shape CLAUDE.md
§"Substring assertions are not structural assertions" rules out.

### AC-3 — The ritual's measurement step precedes its sweep step

The ritual's steps carry their order and the sweep carries its gate. Measuring
factual claims by running commands, and challenging each criterion for the
failure it prevents, are cheap and run every time; the sweep is the expensive
part and runs when the spec asserts a count or an enumeration — which is what
both specs in the motivating episode did.

A finding the sweep produces is a hypothesis until a command settles it. The
ritual says so where a reader acts on a finding, not only in a preamble.

## Constraints

- **The kernel does not change.** No check rule, finding code, config field or
  schema entry ships with this milestone.
- **Authoring lands under `internal/skills/embedded-rituals/plugins/wf-rituals/`.**
  `.claude/skills/` is materialized by `aiwf update` and is not an authoring
  location. The materialized copy in a given clone can be stale (G-0504), so
  read the embedded tree when comparing.
- **The ritual ships with its referencing structural test**, per the repo's
  backstop policy — `wf_structural_sweep_test.go` is the precedent shape. This
  is a constraint rather than an acceptance criterion: "X is tested" is not
  observable behavior.
- **`wf-rituals` stays kernel-agnostic.** The plugin speaks the language of the
  work without coupling to aiwf's planning kernel; an aiwf-specific step belongs
  in `aiwf-extensions`, which is M-0309's side.

## Design notes

- The three parts are not co-equal and the ritual says so. Measurement caught
  every defect in the motivating episode on its own; the criterion-challenge
  killed one criterion; the sweep is where the cost and the one failure sit.
- The criterion-challenge is `wf-vacuity`'s move applied to a specification
  rather than a test suite. Borrow its probe vocabulary rather than minting a
  second one.
- Disposal follows E-0081 and D-0054: a fact the sweep finds stated across
  several surfaces gets an owner and derivations, not a correction per copy.

## Surfaces touched

- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/` — the new ritual
- `internal/policies/` — its referencing structural test

## Out of scope

- Wiring the ritual into any seam. `aiwfx-start-milestone`, `wf-patch` and
  `aiwfx-plan-milestones` are M-0309.
- A check rule over the record. E-0085 defers it until use shows what the record
  contains.
- The structural direction from `quality-signal-and-cadence.md`'s Q6 — requiring
  a criterion to state its own falsification condition.

## Dependencies

- None. First milestone of E-0085.

## References

- E-0085 — parent epic
- G-0583 — the gap the epic closes
- G-0541, M-0307 — the measured episode
- G-0504 — materialized artifacts drift invisibly; read the embedded tree
- D-0054 — keep the reasoning, derive the facts

## Work log

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
