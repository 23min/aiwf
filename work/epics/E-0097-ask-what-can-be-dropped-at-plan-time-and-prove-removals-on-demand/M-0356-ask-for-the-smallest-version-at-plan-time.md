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
      title: plan-milestones drops a candidate no success criterion requires
      status: met
    - id: AC-3
      title: A planning exchange invoking no ritual produces the cut list
      status: cancelled
    - id: AC-4
      title: A planning exchange invoking no ritual gates each addition on a goal
      status: cancelled
---

## Goal

Give the milestone-planning ritual a way to remove work while a cut still costs
nothing: a sizing arm that drops a candidate no success criterion requires, and a
gate that puts the cuts in front of the user before any id is allocated.

## Closes

- (none)

## Context

Both planning rituals confirm a plan rather than challenge it. `aiwfx-plan-epic`
spells scope back and asks for a yes. `aiwfx-plan-milestones` sizes each candidate
with three arms — keep, split, fold into a sibling — and none of the three drops
work, which is the shape [`growth.md`](../../../docs/design/growth.md) names as the
growth mechanism. A plan produced through the epic ritual shrank visibly when the
maintainer asked afterwards whether it was KISS and YAGNI.

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

**Cancelled.** The step this criterion claims was removed after measurement showed it produced nothing the always-on rule did not; see `## Validation`.

### AC-2 — plan-milestones drops a candidate no success criterion requires

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

**Cancelled.** The rule this criterion observed was reworded, so its observation no longer evidences the shipped text; M-0356 AC-4 replaces it. See `## Validation`.

### AC-4 — A planning exchange invoking no ritual gates each addition on a goal

A planning exchange that invokes no ritual arrives already split: a smallest
version that solves the stated problem, and beyond it, additions each carrying
the goal it must serve before it goes in. **Pass criterion**: a recorded exchange
in a session with the guidance installed and no planning skill invoked, where the
smallest version is distinguished from what exceeds it and each addition names
its gating goal; the record holds command, expectation, observation and
environment, including the host and the installed guidance revision. The
criterion claims the split and the gate, not a removal — no arm across three
wordings produced one, and claiming otherwise would assert what was measured
false. **Edge cases**: a plan whose every part serves a stated goal, which says
so rather than manufacturing an addition to gate; a request for one change, which
is not a plan and does not trigger the rule. **Code references**: the rule in
`internal/skills/embedded-guidance/aiwf-guidance.md`.

**Cancelled.** The always-on rule this criterion observed was cut; see `## Validation`.

## Constraints

- Shipped text carries no aiwf id, no path into this tree, and no rationale.
- The cuts the gate names come from the list as drafted, never invented beside it.
- The gate's list is a disclosure the human grades, never a verdict that a plan is
  minimal — G-0585 records why a clearance earned by reading is worse than none.
- Each criterion here claims an observation, so its evidence is the record of
  command, expectation, observation and environment rather than an assertion.

## Design notes

- **Retirement.** The sizing arm stays only while it does something nothing else
  does. At the E-0097 wrap, send prompt 2 under `## Validation` to one repository
  materialized from this milestone's base revision, `2721c11ec`, and one from its
  final revision, in fresh sessions under the same host and model, and record both.
  The two revisions differ in the sizing arm and its gate alone. If the final
  revision no longer drops the scope item no success criterion requires, or the base
  revision drops it too, the arm and its gate are removed.
- D-0070 forbids pinning a phrase in shipped prose, which is why no criterion here
  asserts the presence of the text it adds.

## Surfaces touched

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

The milestone-planning ritual can now remove work, not only reshape it. Its sizing
rule gains a fourth arm — a candidate no success criterion requires is dropped,
where before it could only be kept, split or folded into a sibling — and a gate
that puts the cuts in front of you, and asks for a yes, before any id is allocated.

## Decisions made during implementation

Each is recorded where its reasoning lives rather than restated here.

- The epic-planning step was removed and M-0356 AC-1 cancelled, because the step
  produced nothing the always-on rule did not — `## Validation`, and the removing
  commit in `aiwf history M-0356/AC-1`.
- The always-on rule was cut, and M-0356 AC-3 and M-0356 AC-4 cancelled with it. It
  presented plans split into a smallest version and what exceeds it, removed nothing
  on its own in any run, and its shipped wording was never itself observed —
  `## Validation`.

## Validation

### Method

Each criterion claims an observation of a live session, so each was pre-registered
— prompt, expected outcome, and what would falsify it — and committed before its
run; the pre-registration commits are in `aiwf history M-0356`.

Arms are consumer repositories created by `aiwf init --no-prompt` in an empty git
repository, each from a binary differing only in the surface under test, none
containing aiwf's own source or history. The released arm used the installed
release. Observers are fresh interactive sessions started inside each arm's
repository, so each loads that arm's guidance; every session received
byte-identical text for its prompt and was asked afterwards which files it had read.
Dispatching agents into this repository cannot isolate arms: a dispatched agent
loads its parent session's instructions rather than those of the directory it is
sent to, and an agent inside the implementing repository finds the change in git
and complies conspicuously.

Environment: the Linux development container, 2026-09-26. Asked afterwards, the two
sessions that answered prompt 2 reported model `claude-opus-5-5[1m]` and Claude
Code 2.1.283. The default model changed during the observation window, and a
session reports the model it is running when asked, so which model answered the
prompts is not settled; the version is the binary on PATH, which both sessions
assumed was the one running them.

| arm | always-on rule | epic-planning step | sizing arm |
|---|---|---|---|
| released | — | — | — |
| candidate | first wording | present | present |
| rule without step | first wording | — | present |
| reworded | goal-per-part wording | — | present |
| reworded, labelled | goal-per-part, naming KISS and YAGNI | — | present |

1. *"We need structured audit logging — who changed what, and when — across the
   service. Plan an epic for it. Don't create or commit anything; show me what
   you'd create and stop."*
2. Sent to the same session, verbatim:

   ```
   Break this epic into milestones. Don't create or commit anything; show me the milestone list you'd create and stop.

       Goal
       Every request to the public API is rate limited per client, so one client
       cannot exhaust capacity for the rest.

       Scope
       - Count requests per client over a sliding window.
       - Reject over-limit requests with a retry time.
       - Enforce a configured per-client share.
       - Publish a written guide on choosing limits for new endpoints.

       Success criteria
       - [ ] A client exceeding its limit receives a rejection naming when it may retry.
       - [ ] No client can consume more than its configured share of capacity.
   ```

   The guide serves neither success criterion.
3. *"How should we approach adding rate limiting to the public API? Give me your
   plan."*

### Observed

| arm | prompt | arrived split | outcome |
|---|---|---|---|
| released | 1 | no | one version |
| released | 2 | — | every item kept, split or folded; the guide folded in while the session noted no criterion required it |
| released | 3 | no | a full programme, nothing questioned |
| candidate | 1 | yes | a cut list whose one cut was conditional |
| candidate | 2 | — | the guide dropped: "no success criterion requires it, so by the planning rule it goes" |
| candidate | 3 | yes | an ordered cut list, nothing dropped |
| rule without step | 1 | yes | the candidate's shape; cheapest cut named, then kept |
| reworded | 3 | yes | additions returned as conditions, nothing dropped |
| reworded, labelled | 3 | yes | additions gated on a goal "before it goes in", one defaulted out, the whole plan questioned against an existing gateway; nothing dropped outright |

**M-0356 AC-2 — met.** Both arms read the milestone-planning ritual. Both detected
that the guide served no criterion; only the candidate had a disposition that
removed it, and it cited the ritual's rule by name. The candidate also carried the
always-on rule and the epic-planning step, so the arm contrast alone does not
separate the sizing arm from them; the attribution to the arm rests on the
session's own account. The criterion's second clause — no id allocated for the
dropped candidate — held only because no arm was permitted to allocate anything, so
the gate's ordering was not observed.

**M-0356 AC-1 — cancelled.** The candidate carried the step and did not produce the
one thing the step alone specifies, the smallest version ahead of the body, which
falsifies it on the pre-registered arm alone. A third arm, added afterwards and
differing from the candidate only in the step's absence, produced the same shape.

**M-0356 AC-3 and M-0356 AC-4 — cancelled with the always-on rule.** Six proposals
carried the rule. The five on prompts 1 and 3 all arrived split, against neither
released proposal; the three on prompt 3, where no ritual was read, all split. None
of the five removed anything. The one removal among the six was the candidate's
prompt-2 drop, where the sizing arm applied. The two wordings tested produced no
observable difference; the labelled variant alone questioned the plan as a whole,
in one run. The shipped wording was a compression of the labelled variant and was
never itself observed.

**Limits.** One run per arm and prompt, none repeated, so the spread is unknown.
Reads and the environment are self-reported. The released planning session never
opened the epic-planning ritual, which is why the third arm was needed.

### Measured — what an edit to shipped prose costs

Every edit to the always-on rule — adding it, rewording it, removing it — touched
the guidance fragment, the six frozen host-artifact inventories and the README
section recording their exception; adding it also raised the fragment's line
budget, and removing it restored it. No such edit can be undone with a plain revert:
each rewrites and re-sorts the same inventory lines, so reverting one conflicts with
any edit made after it, and the removal has to be rebuilt forward. Measured in the
Linux development container on 2026-09-26.

### Gate results at wrap

Run in the Linux development container on 2026-09-26, on the milestone branch,
after the always-on rule was removed.

```
go build ./...        → ok
make check-fast       → exit 0; 73 packages ok, 0 failures
aiwf check            → 0 errors, 1 warning
```

The warning is `provenance-untrailered-scope-undefined`: the branch has no
upstream, so the provenance audit has no range to walk and declines rather than
passing. It clears on push, or with `aiwf check --since` naming a ref.

## Deferrals

- (none)

## Reviewer notes

- **Declined: removing the scope-adjustment clause from `aiwfx-plan-milestones`
  step 9.** The pre-allocation gate cannot reach a scope problem found while the
  milestone bodies are written, which happens after allocation; the clause is the
  only place that asks. The gate's own ordering was not observed, so the case for
  removing the later check rests on an assumption rather than a demonstration.
