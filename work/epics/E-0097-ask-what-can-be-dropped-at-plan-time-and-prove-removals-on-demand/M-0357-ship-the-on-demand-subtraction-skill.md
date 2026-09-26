---
id: M-0357
title: Ship the on-demand subtraction skill
status: draft
parent: E-0097
tdd: advisory
acs:
    - id: AC-1
      title: The skill runs end to end on a real diff and its report holds every part
      status: open
    - id: AC-2
      title: Every proposed removal is settled by a command that goes red
      status: open
    - id: AC-3
      title: Each tool the per-stack table names for this stack resolves
      status: open
---

## Goal

Ship a stack-neutral skill that asks of a diff or a named unit whether the change
needs everything it adds, and settles every removal it proposes by a command rather
than by reading.

## Closes

- (none)

## Context

The procedure exists today only as an inline block in the milestone wrap, where the
patch ritual and consumer repositories cannot reach it. The skills around it each
default to adding, strengthening or keeping; none defaults to removal, and the code
review checklist explicitly guards against it. The design, and two hand-run trials
with what each got wrong, are in
[`subtraction-review-on-demand.md`](../../../docs/initiatives/archive/subtraction-review-on-demand.md).

## Acceptance criteria

### AC-1 — The skill runs end to end on a real diff and its report holds every part

The skill runs over a real diff and returns a report holding the verdict that counts
cuts the scope blocked, the behaviour changes awaiting approval with the records
that state the old behaviour, the keep-or-remove guard table with per-guard
evidence, and the handoffs to the test-sufficiency, design and whole-codebase
lenses. **Pass criterion**: a recorded trial naming the diff and the command, where
each part of the report is present or explicitly stated absent with its reason.
**Edge cases**: a diff with no logic bucket at all, which produces a stated skip
rather than an invented finding; a diff where the scope blocks a cut, which the
verdict counts rather than omits. **Code references**: the skill body under the
embedded ritual tree; the trial record lands in this spec's `## Validation`.

### AC-2 — Every proposed removal is settled by a command that goes red

Each removal the skill proposes carries a command whose output shows something going
red when what the removed thing protected is broken. **Pass criterion**: in the
recorded trial, every proposed removal carries its command and that command's
output; a proposal resting on a green gate run fails this criterion. **Edge cases**:
a removal whose protected state no caller can produce, where the evidence is the
demonstration that none can rather than a red run; a stack with no mutation harness,
where the manual probe stands in and the report says so. **Code references**: the
break step in the skill body; G-0660 records the oracle this follows.

### AC-3 — Each tool the per-stack table names for this stack resolves

Every tool the per-stack table names for this repository's stack exists. **Pass
criterion**: a check resolving each command the table names for Go — the mutation
harness, the coverage profile and the clone detector — against this repository, and
failing when one names something absent. **Edge cases**: a tool that exists but is
switched off in configuration, which resolves and is still reported unavailable; a
stack row with no tool for a slot, which names its fallback rather than leaving the
slot blank. **Code references**: a new check under `internal/policies/`; the table in
the skill body.

## Constraints

- The skill states each procedure once and points at the skills that own the others
  rather than restating them.
- Consumer-scoped: imperative instruction only — no aiwf id, no path into this tree,
  no development history and no rationale.
- Stack-neutral: every step named by its method, with a per-stack table naming the
  mutation harness, the coverage profile and the clone detector.
- No aiwf verb is required. Where aiwf is present, the gap and decision recorders are
  named as options, not dependencies.
- A question terminates on a disposition or on a command's result, never on a
  reading (G-0585).
- **The tool-table check draws its needle from the skill body, never from the test.**
  The shipped-prose ban fires only on a needle tracing back to string literals in the
  test source, so a check extracting the tool names from the document and resolving
  each against the repository sits outside the rule and needs no exemption entry.
  Locating the table by its heading does not: that is a test-authored needle against
  shipped content, the class D-0070 retires by name. Derive the candidates from the
  document without naming a section.
- It advises and never blocks: no gate, no commit refused.

## Design notes

- [`subtraction-review-on-demand.md`](../../../docs/initiatives/archive/subtraction-review-on-demand.md)
  holds the twelve-step sketch and the defects both trials found in the trims
  themselves; build from it rather than re-deriving.
- D-0070 sets the evidence form: a relationship check where one exists, a recorded
  trial otherwise.
- **Retirement trigger**: the skill is retired if a re-run over recent patches and
  milestones finds nothing the reviews that already ran did not.

## Surfaces touched

- A new skill under `internal/skills/embedded-rituals/plugins/wf-rituals/skills/`
- `internal/policies/` — the tool-table resolution check

## Out of scope

- Wiring into the wrap rituals, and the skip threshold.
- Whole-codebase discovery, which `wf-structural-sweep` owns, and design rebuild,
  which `wf-rethink` owns; this skill recommends them.
- Applying any rewrite without approval.

## Dependencies

- (none)

## References

- [`docs/initiatives/archive/subtraction-review-on-demand.md`](../../../docs/initiatives/archive/subtraction-review-on-demand.md)
- D-0070 — the limits on pinning shipped prose.
- G-0585 — rituals that clear a question by reading it.
- G-0660 — the repaired oracle for a proposed removal.
- G-0533, G-0253 — the clone detector switched off over the test corpus, and
  statement-scoped coverage; both are conditions the skill's steps meet in this repo.

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
