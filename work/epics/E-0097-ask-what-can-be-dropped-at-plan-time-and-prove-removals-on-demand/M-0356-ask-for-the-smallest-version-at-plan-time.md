---
id: M-0356
title: Ask for the smallest version at plan time
status: in_progress
parent: E-0097
tdd: none
acs:
    - id: AC-1
      title: plan-epic offers the smallest version and argues the delta
      status: cancelled
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

### Withdrawn prompt set

An earlier prompt set was fixed for a design that dispatched fresh-context agents
into this worktree. That design cannot isolate the arms, for two reasons both
observed rather than reasoned: a dispatched agent loads the parent session's
project instructions rather than those of the directory it is pointed at, so the
guidance it follows is not the arm's guidance; and an agent working inside the
repository that implements a rule locates the rule in git and complies with it
conspicuously — one diffed the ritual against the trunk, another read the commit
that added it. No arm was validly exercised under that set, so the prompts are
re-fixed below rather than revised mid-experiment.

Three runs were taken under it and are kept as what they support: with no prompt
asking for a cut, each produced smallest-version framing and a cut list with
reasons, one recommending no milestones at all. That shows the rules are
followable and produce the intended shape. It does not show they fire unprompted,
which is what the criteria claim.

### Pre-registration — written before any run

Each criterion here claims an observation, so the prompts and the expected
outcomes are fixed before either arm is exercised.

**Arms.** Two consumer repositories, each created by `aiwf init` — one from the
released binary, one from a binary built from this milestone's source. Verified
before any run: the released arm carries none of the cut-list rule, the
smallest-version step or the dropping arm; the candidate arm carries all three;
neither contains aiwf's own source or history, so neither can diff the change or
find the commit that made it. This is the view a consumer has, which is the view
the criteria are about.

**Observers.** One live interactive session per arm per prompt class, started in
that arm's repository so it loads that arm's guidance. A planning session and a
no-ritual session are separate within an arm, because a session that has invoked
the planning ritual can no longer serve as the no-ritual observation. Each arm
receives byte-identical prompt text. No session is told an experiment is running,
and none may run a command that creates or commits.

**Prompt 1 — for the epic-planning criterion.** *"We need structured audit
logging — who changed what, and when — across the service. Plan an epic for it.
Don't create or commit anything; show me what you'd create and stop."*

Expected released: the exchange presents one version of the work.
Expected candidate: it presents the smallest version that solves the problem and,
separately, what the larger version adds, with a reason per addition.

**Prompt 2 — for the milestone-decomposition criterion**, sent to the same
session. An epic for per-client rate limiting whose success criteria are a
rejection naming when to retry, and no client exceeding its configured share. Its
scope carries four items, of which three serve those criteria and one — a written
guide on choosing limits for new endpoints — serves neither.

Expected released: every item becomes a candidate, kept, split or folded; none is
dropped, and no cut list precedes allocation.
Expected candidate: the guide is named as a drop because no success criterion
requires it, and the cut list is put up before any id would be allocated.

**Prompt 3 — for the criterion covering a plan with no ritual.** *"How should we
approach adding rate limiting to the public API? Give me your plan."* No skill is
named and no ritual is invoked.

Expected released: a plan arrives with no cut list.
Expected candidate: the plan distinguishes the smallest version from what exceeds
it, and names what it would drop.

**What would falsify each.** A candidate-arm run producing the expected shape only
because the prompt asked for it; a released-arm run already producing it, which
would mean the change is not what causes the difference; or a decomposition whose
drop is justified on grounds other than no criterion requiring it, which would
leave the new sizing arm unexercised. Each is reported as observed rather than
reconciled.

### Observed

Five runs, one per arm per prompt, each a live session started in that arm's
repository and sent the prompt text above unchanged. Every session was asked
afterwards which files it had read; the attributions below rest on those answers.

**The criterion covering a plan with no ritual — met.** Neither arm invoked a
skill, so the criterion's precondition holds. The released arm returned three
milestones, decisions to record, metrics, a shadow rollout and a published
contract, with no cut list anywhere. The candidate arm returned a smallest
version, what the larger one adds with a reason each, and an ordered cut list
that also named what could not be cut and why. It attributed that to the rule by
name: *"'A plan arrives with its cut list' gave the smallest version, the larger
additions and the cut-first order."*

**The criterion covering milestone decomposition — met.** Both arms read the
decomposition ritual, so exposure was symmetric and only its text differed. The
released arm kept, split or folded every scope item and dropped none, folding the
unserved guide into a sibling. It identified the condition itself — *"The written
guide has no success criterion. Neither criterion would fail if the guide never
shipped"* — and could offer only to add a criterion or accept the deliverable
unchecked. The candidate arm dropped it: *"no success criterion requires it, so by
the planning rule it goes."* Both arms detected the surplus; only one had a
disposition that removed it.

**The criterion covering the epic-planning step — cancelled, its claim
disconfirmed.** A third arm was materialized carrying the always-on rule but not
the step, differing from the candidate in that alone. It opened the planning
ritual, found no step there, and produced the same shape the candidate did,
attributing it to the rule. The ordering the step alone specifies — the smallest
version ahead of the body — appeared in neither arm. The step was removed rather
than left unevidenced.

**What the arms did not show.** The rule produces a cut list reliably; it produced
a cut once in four. Two arms carrying it named a cheapest cut and then kept it, and
one returned a conditional for the reader to settle. The single unconditional
removal came from the decomposition ritual's sizing arm, which tests an item
against the epic's success criteria, rather than from the rule, which asks a
drafter to judge its own draft. On this evidence a rule carrying a test removes
work, and a rule asking for a judgement produces the list and stops.

**Limits.** One run per arm, five in total, none repeated, so the spread is
unknown. The pattern is consistent across every run and matches the operating
experience that prompted this work, but five runs is a signal rather than a
measurement. The released arm never opened the epic-planning ritual, which is why
the original two-arm design could not have attributed anything to that step, and
why a third arm was needed.

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
