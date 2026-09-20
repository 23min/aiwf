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
      title: The growth report is re-run against the pre-epic baseline and its row logged
      status: open
---

## Goal

Run the rubric again on the shrunk guidance and re-run the growth report against the post-delivery, pre-reduction baseline, so the epic's claim is judged against a prediction rather than read for a story.

## Closes

- (none)

## Context

M-0334 fixes the post-delivery, pre-reduction baseline, rubric, source revision and host settings. Compare each host with its own baseline at M-0337's completion. Keep installed external guidance and personal overlays unchanged so the result measures this epic's reduction. The outcome determines whether M-0339 runs.

## Acceptance criteria

### AC-1 — The after observation is recorded against the same rubric

Repeat M-0334's tasks and run counts for both hosts at M-0337's completion, judged against the unchanged rubric. Record commands/prompts, expectations, observed reads and behavior, versions, checkout and guidance revisions. Compare each host against its own baseline, including root-started nested work, new files and post-compaction continuation. Unavailable observations remain outstanding. The fragment stage requires no observed lost effect in either host.

### AC-2 — The growth report is re-run against the pre-epic baseline and its row logged

Run the report at HEAD and at M-0334's frozen post-delivery, pre-reduction revision. Append the commands and results to the growth iteration log. Compare upfront counts per host, conditional inventory, observed task-loaded text and policy/test growth against the baseline expectations; name regressions. Do not use the pre-delivery snapshot as this comparison's baseline.

## Constraints

- No cut lands between M-0337's last commit and the after-run.
- The rubric is used as committed; a needed change is recorded as a finding here, not applied.
- Every figure carries its command (G-0668).

## Design notes

- State the no-lost-effect judgment separately for each host; it is bounded by the recorded task set, not a universal compliance guarantee.
- Correct lost routing or weakened rules at their canonical source and repeat affected observations before permitting the fragment stage.

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
