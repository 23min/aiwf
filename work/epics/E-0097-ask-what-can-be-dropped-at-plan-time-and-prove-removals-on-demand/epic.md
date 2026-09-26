---
id: E-0097
title: Ask what can be dropped at plan time and prove removals on demand
status: proposed
---
## Goal

Put the subtraction question at the two moments it pays: before work is proposed,
where cutting is free, and on demand over a change or a unit, where removing what
exists has to be proved. The planning rituals gain a step that offers the smallest
version first; a shipped skill gains the procedure for removing what already
landed, and the wrap rituals call it instead of restating it.

## Context

Overbuilding is prevented by a question asked at a moment, not by a principle
stated again. KISS and YAGNI are primed in this repository in the root
instructions, in the always-on fragment's economy priming, and in the personal
guidance loaded beside them; a planning session under all of it still produced a
plan that a single question from the maintainer visibly shrank. The question
worked because it arrived at a named moment and had to be answered.

Neither planning ritual asks it. `aiwfx-plan-epic` confirms scope by spelling it
back, which ratifies a plan rather than challenging it, while its own anti-pattern
list states the economics — the epic spec is where scope changes are cheap.
`aiwfx-plan-milestones` sizes each candidate with three arms: keep, split, or fold
into a sibling. None of the three drops work, which is the shape
[`growth.md`](../../../docs/design/growth.md) names as the growth mechanism — a rule
set where every member can mandate and none can retire.

Downstream of planning, the only lens that asks whether something is surplus is the
milestone wrap's shape block. `wf-patch` asks none of its questions (G-0662), so
work that never becomes a milestone never meets it. The block is also the one copy
of a procedure that belongs in a skill, where the patch ritual and any consumer
could reach it.

[`subtraction-review-on-demand.md`](../../../docs/initiatives/subtraction-review-on-demand.md)
holds the on-demand skill's design and two hand-run trials, including what each
trial got wrong and what that asks of the skill.

## Scope

- The cut-list clause in the always-on guidance source, so the question reaches a
  plan that invokes no ritual.
- Plan-time subtraction in both planning rituals: offer the smallest version and
  argue the delta; add the sizing arm that drops work; put the cut gate ahead of id
  allocation, while a cut is still free.
- A shipped, stack-neutral subtraction skill that runs on a diff or a named unit,
  settles every removal by a command, and hands its by-products to the skills that
  own them.
- Wiring: the milestone wrap calls the skill in place of the four shape questions
  it holds inline, keeps the one that names what a change obliges the future to do,
  and the patch ritual gains the lens behind a stated threshold.

## Out of scope

- **Build-time prevention** — the commit-gate question naming what a change adds
  beyond the task, and attrition in the review dispositions. Plan time is upstream
  of both; revisit once this epic's step has been observed.
- **Any gate, budget or metric over plan-time subtraction.** `growth.md` rules it
  out directly: a growth budget enforced by a chokepoint is the same mistake one
  level up.
- **A spec-template section for dropped work.** A template seeding a heading and a
  ban forcing it to be filled compose into an obligation neither one is (G-0530).
- **Generalizing the guidance commit-seam fence** to other artifact classes. It is
  repo-only by construction, so it never reaches a consumer, and one measured
  regrowth record does not justify a second fence.
- **Whole-codebase discovery**, which `wf-structural-sweep` owns, and **design
  rebuild**, which `wf-rethink` owns. The new skill recommends them; it does not
  absorb them.
- Changing what `wf-review-code` or `wf-vacuity` default to.

## Constraints

- **A question terminates on a disposition or on a command's result, never on a
  reading.** G-0585 records why: a clearance earned by reading is worse than none,
  because the reader holding it stops looking. The cut list is a disclosure the
  human grades, never a verdict that a plan is minimal.
- **A removal is settled by breaking what it protected and watching something go
  red**, not by a green gate — the oracle G-0660 repaired for the compression lens,
  carried into every port of it.
- **Every addition this epic makes names what retires it.** An addition that cannot
  say what retires it is a permanent tax, and an epic against overbuilding that
  ships one has failed its own test.
- **The drop candidates come from the proposal as drafted**, never invented extras.
  This is what makes a padded list expensive rather than free.
- **Shipped text is consumer-scoped**: no real ids, no paths into this tree, no
  development history or rationale. `skill-body-id` blocks the push, and E-0096
  widens that gate.
- **Evidence is observation-class where no relationship check exists.** D-0070
  forbids pinning a phrase in shipped prose, and these deliverables are prose, so
  each records command, expectation, observation and environment instead.
- **The cut-list clause lands in the always-on guidance source ahead of E-0092's
  reduction, and is registered with that epic's rule inventory as already placed.**
  Its audience decides its home, by that epic's own test: offering the smallest
  version first is how to operate in any repository, not how to develop aiwf. It is
  the class of rule that must fire without anyone reaching for a skill, because the
  decision to overbuild is taken before any ritual is entered.
- **The shape questions divide by what they measure.** The four that measure a
  change's lines — what was retired, whether same-outcome tests fail for distinct
  reasons, whether the logic compresses, whether each guard has a caller — belong to
  the skill, and each needs a command to settle. What a change obliges every later
  change to do stays with the wrap: it must be answerable where there is no logic to
  compress and no guard to justify, and its answer belongs in the milestone's own
  record. The skill may report an obligation it finds; naming that obligation's owner
  and what retires it remains the wrap's requirement, so one observation yields two
  outputs rather than two copies.
- **A skipped lens is stated where the human can veto it**, with its reason, in the
  form the patch ritual already uses for its review carve-out.
- Every edit under the embedded ritual and guidance trees rides a commit carrying
  an `aiwf-entity` trailer.

## Success criteria

- [ ] A planning session on a real candidate offers the smallest version that
      solves the stated problem, states what the proposed version adds beyond it,
      and records what was dropped — before any id is allocated.
- [ ] The milestone sizing rule has an arm that drops work.
- [ ] The subtraction question reaches a plan that invokes no ritual.
- [ ] The subtraction skill runs on a diff and on a named unit, in a repository
      that does not use aiwf, and every removal it proposes is settled by a command
      whose output it reports.
- [ ] The shape questions exist in one place: the milestone wrap calls the skill,
      and no ritual restates the procedure.
- [ ] The patch ritual asks the shape questions, with the skip threshold stated
      where a reader meets it.
- [ ] Every obligation this epic adds names what retires it.
- [ ] G-0662 is addressed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| The threshold below which the patch wrap skips the lens | yes, before the port | Decided in the port, from the logic changed or the guards added |
| How large a diff one agent takes before it goes shallow | no | The milestone wrap already slices by concern; reuse its rule |
| What the skill does in a stack whose mutation harness or clone detector is missing | no | The per-stack table records the absence and the fallback |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The cut list becomes theatre — a list padded with candidates never intended to be built | high | Candidates are drawn from the drafted proposal, so a reader sees whether the list names what is on the page |
| The model grades its own plan | high | The list is a disposition presented to the human, who decides; no clearance vocabulary (G-0585) |
| The skill becomes a mandate on every change and outgrows what it saves | high | A stated threshold and a named retirement trigger, both read at review as the epic's own test |
| A guidance edit confounds E-0092's before-and-after comparison | med | The window question above is settled before the clause is written |
| The planning step ratifies rather than challenges, as the spell-back it replaces did | med | The smallest version is the proposal and the delta carries the argument, so silence produces the smaller plan |

## Milestones

- `M-0356` — the cut-list clause and the step in both planning rituals · depends on: —
- `M-0357` — the on-demand subtraction skill · depends on: —
- `M-0358` — the wrap and patch-ritual wiring · depends on: `M-0357`

## References

- [`docs/initiatives/subtraction-review-on-demand.md`](../../../docs/initiatives/subtraction-review-on-demand.md) — the on-demand skill's design and its two hand-run trials.
- [`docs/design/growth.md`](../../../docs/design/growth.md) — the measured baseline, the four-shape model, and the rule against gating a growth budget.
- D-0054 — keep the reasoning, derive the facts; record obligations, not events.
- D-0070 — the limits on pinning shipped prose, which set this epic's evidence form.
- G-0662 — the patch ritual asks none of the wrap's shape questions.
- G-0585 — rituals that clear a question by reading it.
- G-0660 — the repaired oracle for a proposed removal.
- G-0530 — a template and a ban composing into a mandate.
- G-0533, G-0253 — the duplication detector switched off over the test corpus, and statement-scoped coverage, both of which the skill's steps meet.
- E-0092 — the guidance epic that owns the always-on source while it runs.
- E-0096 — the gate over everything shipped, which this epic's new surfaces must pass.
- G-0698 — the ritual-branch requirement in the planning rituals, adjacent and untouched here.
