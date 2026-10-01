---
id: M-0338
title: Record the after observation and re-run the growth report
status: draft
parent: E-0092
depends_on:
    - M-0337
tdd: none
acs:
    - id: AC-1
      title: The after observation is recorded against the same rubric
      status: open
    - id: AC-2
      title: Growth is compared with the frozen post-delivery baseline
      status: open
    - id: AC-3
      title: Every inventory decision has been carried out
      status: open
---

## Goal

Run the rubric again on the shrunk guidance, re-run the growth report against the post-delivery, pre-reduction baseline, and confirm every inventory decision was carried out, so the epic's claim is judged against a prediction rather than read for a story.

## Closes

- (none)

## Context

M-0334 fixes the post-delivery, pre-reduction baseline, rubric, source revision and host settings. Compare each host with its own baseline at M-0337's completion. Keep installed external guidance and personal overlays unchanged so the result measures this epic's reduction; `guidance.enabled` stays `false` until this milestone completes.

## Acceptance criteria

### AC-1 — The after observation is recorded against the same rubric

Repeat M-0334's tasks and run counts for both hosts at M-0337's completion, judged against the unchanged rubric. Record commands/prompts, expectations, observed reads and behavior, versions, checkout and guidance revisions. Compare each host against its own baseline, including root-started nested work, new files and post-compaction continuation, and every rule M-0349's inventory changed: moved, merged, tightened or replaced with a pointer. Confirm that each host now reaches on-demand guidance through `.guidance/project.md` for the tasks that need it. Unavailable observations remain outstanding.

### AC-2 — Growth is compared with the frozen post-delivery baseline

Run the report at HEAD and at M-0334's frozen post-delivery, pre-reduction revision. Append the commands and results to the growth iteration log. Compare upfront counts per host, conditional inventory, observed task-loaded text and policy/test growth against the baseline expectations; name regressions. Do not use the pre-delivery snapshot as this comparison's baseline.

### AC-3 — Every inventory decision has been carried out

Every row of M-0349's inventory whose decision changes text has a commit that carried it out and names the row in its disposition block. **Pass criterion**: M-0336 AC-5's test, run with every decision required, reports a row with no carrying commit; on the tree at this milestone's end it reports none. A row left undone is carried out, or changed by a maintainer decision recorded in M-0349, before the epic wraps.

## Constraints

- No cut lands between M-0337's last commit and the after-run.
- The rubric is used as committed; a needed change is recorded as a finding here, not applied.
- Every figure carries its command (G-0668).

## Design notes

- State the no-lost-effect judgment separately for each host; it is bounded by the recorded task set, not a universal compliance guarantee.
- Correct lost routing or weakened rules at their canonical source and repeat affected observations before the epic wraps.

## Surfaces touched

- this milestone's Validation section
- `docs/design/growth.md` §"Iteration log"

## Out of scope

- Any change to guidance.

## Dependencies

- M-0337 — the ceiling at target
- M-0334 — the rubric and the baseline this compares against

## Coverage notes

- (none)

## References

- `docs/design/growth.md`, `scripts/growth-report.py`
- G-0668

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
