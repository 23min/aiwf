---
id: M-0308
title: Ship the spec-measurement ritual and settle what its record is
status: in_progress
parent: E-0085
tdd: required
acs:
    - id: AC-1
      title: A completed pass leaves a record naming the entity whose claims it measured
      status: met
      tdd_phase: done
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

The steps appear in yield order, and the order is a property of the document
rather than of any word in it: measuring factual claims by running commands
comes before sweeping related prose. Reordering the ritual fails this criterion.

The assertion reads the step headings in document order and compares positions.
It does not ask whether a section mentions ordering — a claim like that passes
because someone wrote the word, which is the failure mode AC-2's derivation
avoids and which this criterion avoids by testing position instead of presence.

The scope here is deliberately narrower than "the steps run in yield order with
the sweep gated on countable claims." The gate is a real obligation on the
ritual's content and it appears under `## Constraints`, because no assertion
over prose distinguishes a sweep that carries a genuine precondition from one
whose section merely contains the word. Per CLAUDE.md §"AC promotion requires
mechanical evidence", a criterion with no mechanical form is not a criterion.

## Constraints

- **The kernel does not change.** No check rule, finding code, config field or
  schema entry ships with this milestone.
- **The sweep is gated on the spec asserting a count or an enumeration.** The
  cheap two steps run every time; the expensive one runs when there is something
  countable to falsify, which is what both specs in the motivating episode had.
  This is a constraint rather than a criterion because no assertion over prose
  separates a genuine precondition from a section containing the word.
- **A sweep finding is a hypothesis until a command settles it**, and the ritual
  says so where a reader acts on a finding rather than only in a preamble. Same
  reason it is a constraint: the obligation is real and its mechanical form is
  not.
- **Authoring lands under `internal/skills/embedded-rituals/plugins/wf-rituals/`.**
  `.claude/skills/` is materialized by `aiwf update` and is not an authoring
  location. The materialized copy in a given clone can be stale (G-0504), so
  read the embedded tree when comparing.
- **The ritual ships with its referencing structural test**, per the repo's
  backstop policy. `wf_structural_sweep_test.go` supplies the file layout and
  the section-scoping helpers; its matching is not the model, per the Design
  note below. This is a constraint rather than an acceptance criterion: "X is
  tested" is not observable behavior.
- **An aiwf-specific step in `wf-rituals` is conditional, not absent.** The
  plugin's skills speak the language of the work and guard a kernel step behind
  its precondition — `wf-tdd-cycle`'s *"If the project uses aiwf and the
  milestone is `tdd: required`"* is the house form, and `wf-patch` and
  `wf-doc-lint` name aiwf verbs the same way. What belongs in `aiwf-extensions`
  is a step with no meaning outside aiwf, which is M-0309's side.

## Design notes

- The three parts are not co-equal and the ritual says so. Measurement caught
  every defect in the motivating episode on its own; the criterion-challenge
  killed one criterion; the sweep is where the cost and the one failure sit.
- The criterion-challenge is `wf-vacuity`'s move applied to a specification
  rather than a test suite. Borrow its probe vocabulary rather than minting a
  second one.
- Disposal follows E-0081 and D-0054: a fact the sweep finds stated across
  several surfaces gets an owner and derivations, not a correction per copy.
- CLAUDE.md §"AC promotion requires mechanical evidence" governs every criterion
  here, and names the doc-shaped form: a structural assertion scoped to a named
  markdown section. `extractMarkdownSection` and `countSubHeadings` under
  `internal/policies/` are the house helpers.
- **The existing ritual tests are not the shape to copy.**
  `wf_structural_sweep_test.go` scopes to a section correctly and then matches
  hardcoded literals chosen by reading the prose, so it passes because someone
  typed those words and survives the lens being gutted. Scoping is structural
  there; the matching is not.
- AC-1 and AC-2 derive the expected side from the cobra command tree, walked
  from the root command: the ritual names a command, and the assertion resolves
  it. No verb-name enumeration exists to use instead — `worktree` is a CLI
  command under `internal/cli/worktree/`, not an entry in `internal/verb/`. If
  the walk proves unreachable, the affected criterion demotes to a constraint
  rather than shipping an assertion that only proves the prose contains a
  string.

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

### AC-1 — the record shape, settled then pinned

`wf-measure-spec` ships with a `## The record` section naming `aiwf edit-body`
and two outcomes; its test derives the verb from the cobra tree via
`findAllVerbs` and the outcome count from heading shape, so a rename on either
side goes red · commit 6728317a5 · four mutants probed, none survived

## Decisions made during implementation

- D-0066 — the record is a `## Spec measurement` section in the measured
  entity's body, landed with `aiwf edit-body`. The other two candidates AC-1
  named were eliminated by measurement rather than preference: `edit-body` emits
  no trailer a record could ride, and an empty `edit-body` produces no commit in
  either mode.

## Validation

## Deferrals

- G-0584 — the existing ritual policy tests match section-scoped literals that
  cannot fail. Surfaced while sizing this milestone's own assertions. Out of
  scope here: this milestone writes one new test and touches none of those six
  files, so the cheap-fix test does not carry it.

## Reviewer notes
