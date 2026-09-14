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

Run the rubric again on the shrunk guidance and re-run the growth report against the pre-epic baseline, so the epic's claim is judged against a prediction rather than read for a story.

## Closes

- (none)

## Context

M-0334 wrote the rubric, ran the baseline, and stated the expected direction of each growth metric. M-0337 brought the always-on set to its target. Nothing has changed in the shipped fragment yet, so what this run measures is the repository's own guidance, and its result decides whether M-0339 runs.

## Acceptance criteria

### AC-1 — The after observation is recorded against the same rubric

The same tasks, run the same number of times, in a worktree at the commit that closed M-0337, judged blind against the rubric M-0334 committed. **Pass criterion**: Validation records, per task, the command, the expected count, the observed counts per run, and the environment, beside the baseline figures, with the rubric's commit sha cited. This is a record, met by the record.

### AC-2 — The growth report is re-run against the pre-epic baseline and its row logged

`scripts/growth-report.py` is run at HEAD and at `--at <pre-epic sha>`, and a row is appended to the iteration log in `docs/design/growth.md`. **Pass criterion**: the row carries the command; Validation states, per metric, the observed direction against the expectation M-0334 recorded, and names any that went the other way.

## Constraints

- No cut lands between M-0337's last commit and the after-run.
- The rubric is used as committed; a needed change is recorded as a finding here, not applied.
- Every figure carries its command (G-0668).

## Design notes

- The judgment that decides M-0339 is written here in one paragraph: lost effect, or none. Restoring a rule whose effect was lost is a targeted edit to its one home, recorded as a gap discovered in this epic, not a revert.

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
