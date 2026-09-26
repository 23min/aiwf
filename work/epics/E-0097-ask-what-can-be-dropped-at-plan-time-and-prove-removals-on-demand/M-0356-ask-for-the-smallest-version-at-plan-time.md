---
id: M-0356
title: Ask for the smallest version at plan time
status: in_progress
parent: E-0097
tdd: none
acs:
    - id: AC-1
      title: plan-epic offers the smallest version and argues the delta
      status: open
    - id: AC-2
      title: plan-milestones drops an unrequired candidate before allocating its id
      status: open
    - id: AC-3
      title: A planning exchange invoking no ritual produces the cut list
      status: open
---

## Goal

Put the subtraction question at the moment work is proposed — inside both planning
rituals and in the always-on guidance — so it fires whether or not a ritual is
invoked, while a cut still costs nothing.

## Closes

- (none)

## Context

Both planning rituals confirm a plan rather than challenge it. `aiwfx-plan-epic`
spells scope back and asks for a yes. `aiwfx-plan-milestones` sizes each candidate
with three arms — keep, split, fold into a sibling — and none of the three drops
work, which is the shape [`growth.md`](../../../docs/design/growth.md) names as the
growth mechanism. A planning session run through the epic ritual produced a plan
that one question from the maintainer visibly shrank afterwards, which places the
missing step inside the ritual's own output rather than beside it. The always-on
guidance carries the economy priming and no obligation to produce a cut list.

## Acceptance criteria

### AC-1 — plan-epic offers the smallest version and argues the delta

`aiwfx-plan-epic` presents the smallest version that solves the stated problem
before the proposed one, then states what the larger version adds with an argument
per addition. **Pass criterion**: a recorded planning run on a real candidate whose
exchange carries both, against the same ritual at the prior revision on the same
candidate carrying neither; the record holds the command, the expectation written
before the run, the observed exchange and the environment. **Edge cases**: a
candidate whose smallest version is the proposed one, where the step says so rather
than inventing something smaller; a candidate the user has already constrained,
where the delta is theirs rather than the assistant's. **Code references**: the step
lands in the embedded `aiwfx-plan-epic` ritual body; the record lands in this spec's
`## Validation`.

### AC-2 — plan-milestones drops an unrequired candidate before allocating its id

`aiwfx-plan-milestones` drops a candidate not required by the epic's success
criteria, and the drop happens before `aiwf add milestone` allocates an id for it.
**Pass criterion**: a recorded run over a decomposition containing such a candidate,
where the ritual names it as a drop and no id is allocated for it; the record holds
command, expectation, observation and environment. **Edge cases**: a candidate
required only indirectly, through another candidate it enables; a decomposition
where nothing qualifies, which produces a stated *nothing was droppable* rather than
silence. **Code references**: the sizing arm and the gate land in the embedded
`aiwfx-plan-milestones` ritual body.

### AC-3 — A planning exchange invoking no ritual produces the cut list

A planning exchange that invokes no ritual arrives with its cut list. **Pass
criterion**: a recorded exchange in a session with the guidance installed and no
planning skill invoked, where the proposal carries drop candidates drawn from what
it itself drafted; the record holds command, expectation, observation and
environment, including the host and the installed guidance revision. **Edge cases**:
a proposal with nothing to drop, which states that rather than padding the list; a
request for one change, which is not a plan and does not trigger the list. **Code
references**: the clause lands in `internal/skills/embedded-guidance/aiwf-guidance.md`.

## Constraints

- The clause does not exceed the median length of the fragment's existing top-level
  rules.
- **The clause's commit carries this milestone's `aiwf-entity` trailer, and that is
  the whole of its registration.** The guidance inventory can trace any fragment
  rule to the entity that placed it, so a second record would be a copy.
- **No operating-anchor entry for the new rule.** An anchor is a presence assertion
  over shipped prose, and the anchors policy is rewritten under E-0092.
- Shipped text carries no aiwf id, no path into this tree, and no rationale.
- The drop candidates come from the proposal as drafted, never invented extras.
- The cut list is a disposition the human grades, never a verdict that a plan is
  minimal — G-0585 records why a clearance earned by reading is worse than none.
- Each criterion here claims an observation, so its evidence is the record of
  command, expectation, observation and environment rather than an assertion.

## Design notes

- The clause's home follows E-0092's own audience test; the epic's constraints hold
  the argument.
- **Retirement trigger for the clause**: E-0092's before-and-after rubric. If plans
  do not come out smaller, the rule goes.
- D-0070 forbids pinning a phrase in shipped prose, which is why no criterion here
  asserts the presence of the text it adds.

## Surfaces touched

- `internal/skills/embedded-guidance/aiwf-guidance.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md`

## Out of scope

- The commit-gate question naming what a change adds beyond the task.
- Any gate, budget or metric over plan-time subtraction.
- A spec-template section for dropped work.
- The on-demand subtraction skill and its wiring.

## Dependencies

- (none)

## References

- E-0092 — the guidance epic whose inventory meets this rule.
- D-0054 — record obligations, not events.
- D-0070 — the limits on pinning shipped prose.
- G-0585 — rituals that clear a question by reading it.
- [`docs/design/growth.md`](../../../docs/design/growth.md) — the four-shape model and the rule against gating a growth budget.

## Release note

## Decisions made during implementation

- (none)

## Validation

### Pre-registration — written before any run

Each criterion here claims an observation, so the prompt and the expected
outcome are fixed before either arm is exercised and are not revised afterwards.
Observations are recorded below this section as they are taken.

**Design.** One frozen prompt per criterion, run unchanged against two arms. The
*before* arm is the artifact set this worktree was created with, captured at
`aiwfx-plan-epic/SKILL.md` `5a4cb525…`, `aiwfx-plan-milestones/SKILL.md`
`3502fc53…` — both byte-identical to the records frozen in the host-artifact
compatibility baseline — and the guidance fragment carrying no cut-list rule. The
*after* arm is the same tree re-materialized from a binary built from this
milestone's source. Each run is a fresh-context agent with no knowledge of the
change, dispatched into this worktree. No run may execute a mutating command; the
observable in every case is what the exchange puts in front of the reader before
anything is allocated.

**Prompt 1 — for the epic-planning criterion.** *"Plan an epic for adding a
rehearsal mode to every mutating aiwf verb, so an operator can see what a command
would do without it happening. Do not run any mutating command — present what you
would create and stop."*

Expected before: the exchange spells scope back and presents one version.
Expected after: the exchange presents the smallest version that solves the problem
and, separately, what the larger version adds, with a reason per addition.

**Prompt 2 — for the milestone-decomposition criterion.** The same agent is handed
an epic whose success criteria are stated and which carries one candidate no
criterion requires, and asked to decompose it without allocating.

Expected before: every candidate is kept, split or folded; none is dropped, and no
cut is named before allocation would occur.
Expected after: the candidate no success criterion requires is named as a drop, and
the cut list is put up before allocation.

**Prompt 3 — for the criterion covering a plan with no ritual.** *"How should we
approach making aiwf's finding messages consistent? Give me your plan."* No skill
is named, and the phrasing asks for an approach rather than naming a ritual.

Expected before: a plan arrives with no cut list.
Expected after: the plan arrives with the smallest version distinguished from what
exceeds it, and names what it would drop.

**What would falsify each.** An after-arm run that produces the expected shape only
because the prompt asked for it; a before-arm run that already produces it, which
would mean the change is not what causes the difference. Both are reported as
observed rather than reconciled.

### Measured — the cost of the guidance rule

A five-line rule added to the always-on fragment obligated nine files: the rule,
the fragment's line-budget constant, six frozen host-artifact inventories, and the
exception record their README requires. The budget moved from 169 to 173 and was
set to the exact new size, so the next rule argues for itself rather than using
slack. Measured in the Linux development container on 2026-09-26.

## Deferrals

- (none)

## Reviewer notes

- (none)
