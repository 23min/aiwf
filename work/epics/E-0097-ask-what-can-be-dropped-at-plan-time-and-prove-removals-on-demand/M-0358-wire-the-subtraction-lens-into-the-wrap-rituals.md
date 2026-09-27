---
id: M-0358
title: Wire the subtraction lens into the wrap rituals
status: in_progress
parent: E-0097
depends_on:
    - M-0357
tdd: none
acs:
    - id: AC-1
      title: The skill answers the four shape questions and the wrap asks only the obligation
      status: open
    - id: AC-2
      title: The patch ritual runs the lens above its threshold and states the skip below it
      status: open
    - id: AC-3
      title: The tracked gap's measurement is re-run and no longer holds
      status: open
---

## Goal

Give the four line-measuring shape questions one home by having the milestone wrap
call the skill, keep with the wrap the question of what a change obliges every later
change to do, and give the patch ritual the lens behind a stated threshold.

## Closes

- G-0662 — the patch ritual reaches the shape questions through the lens.

## Context

The milestone wrap holds five shape questions inline — the only questions at wrap
that ask whether something is surplus rather than missing. The patch ritual asks
none of them, which G-0662 records as a measurement against that ritual's body, on
the argument that the patch is the higher-traffic surface. Which questions move and
which stays is settled in this epic's constraints.

## Acceptance criteria

### AC-1 — The skill answers the four shape questions and the wrap asks only the obligation

The milestone wrap obtains the four line-measuring answers from the skill and asks,
itself, only what the change obliges every later change to do. **Pass criterion**: a
recorded wrap on a real milestone where the skill was invoked, its report answered
the four, and the ritual's own question produced the obligation record in the
milestone spec, each named obligation carrying its owner and what retires it; the
record holds command, expectation, observation and environment. **Edge cases**: a
milestone below the skip threshold, where the skip is stated and the obligation
question is still asked; a milestone with no logic, where the four are stated
inapplicable and the obligation question is still answered. **Code references**: the
embedded `aiwfx-wrap-milestone` ritual body.

### AC-2 — The patch ritual runs the lens above its threshold and states the skip below it

The patch ritual runs the lens above its threshold, and below it states the skip and
its reason at the commit gate where the human can veto it. **Pass criterion**: two
recorded patches, one each side of the threshold, the first carrying the lens's
report and the second a stated skip naming the threshold it fell under. **Edge
cases**: a patch with no logic at all, which already carries a review carve-out and
does not acquire a second statement; a patch changing few lines but adding a guard,
which the threshold catches on guards rather than on lines. **Code references**: the
embedded `wf-patch` ritual body.

### AC-3 — The tracked gap's measurement is re-run and no longer holds

G-0662's measurement no longer holds. **Pass criterion**: that gap's own measurement
re-run against the patch ritual's body and reported with its command and output,
showing the shape questions reachable from that ritual where each previously scored
zero. **Edge cases**: a question reachable only through the skill, which counts as
reachable and is recorded as such rather than as present inline. **Code
references**: `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`;
closure rides `aiwf promote G-0662 addressed --by-commit <sha>`.

## Constraints

- **The division is drawn on output.** The skill may report an obligation it finds;
  naming that obligation's owner and what retires it stays the wrap's requirement.
  One observation, two outputs, never two copies.
- No procedure is restated: both rituals call the skill.
- A skipped lens is stated where the human can veto it, with its reason.
- The threshold is set from logic lines changed or guards added, is stated in the
  ritual where a reader meets it, and lands with what retires it.
- Shipped text carries no aiwf id, no path into this tree, and no rationale.

## Design notes

- The epic's constraints hold the argument for which questions move; this milestone
  applies it rather than re-deciding it.
- G-0660's repaired oracle travels with the compression question: a removal is
  never settled by a green run. The skill settles it by something going red, or,
  for a guard, by the demonstration that no caller reaches its state; a cut that
  changes no result is a rewrite, settled by a differential test. The wrap block's
  "Nothing red is a surviving mutant, not a clearance" contradicts the second route
  until the block calls the skill.
- `wf-trim` joins the milestone wrap's independent review as a third lens beside
  code quality and design quality, and the shape block keeps only the obligation
  question. The block's exclusion of mandated comments from half, and its wording
  for the Deletions question, are not carried: where compression cannot reach half,
  the skill names the constraint that stops it, a mandated comment among them, and
  every empty section of its report states why it is empty. The skill is unchanged.
  Calling it also settles a rewrite by a differential test, where the block settles
  one by a green gate run.
- The lens adds no human step. Its report joins the other lenses' findings, and its
  proposals are approved with theirs once every review has returned; the skill's
  second-agent confirmation runs after that approval and before any cut is
  committed. The patch ritual takes the same shape, with its commit gate as the
  approval point.

## Surfaces touched

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`

## Out of scope

- Changing what the skill itself does.
- The commit-gate question naming what a change adds beyond the task.
- Porting the lens to the epic wrap.

## Dependencies

- M-0357 — the skill both rituals call.

## References

- G-0662 — the patch ritual asks none of the wrap's shape questions.
- G-0660 — the repaired oracle for a proposed removal.
- G-0585 — rituals that clear a question by reading it.

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
