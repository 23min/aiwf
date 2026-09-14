---
id: M-0334
title: Write the rubric, record the baseline, add guidance metrics to growth-report
status: draft
parent: E-0092
tdd: none
acs:
    - id: AC-1
      title: The rubric names each judgment rule and what counts as a violation
      status: open
    - id: AC-2
      title: The baseline observation is recorded with its command and environment
      status: open
    - id: AC-3
      title: Growth-report tracks always-on words and CLAUDE.md commit rate
      status: open
---

## Goal

Write the rubric and capture the before picture: how the assistant behaves against the judgment rules over a fixed task set, and the growth metrics at the pre-epic commit.

## Closes

- (none)

## Context

M-0338 compares against what this milestone records, so the rubric has to exist before the first run and not change after. `docs/design/growth.md` and `scripts/growth-report.py` already hold the tree's growth metrics and an iteration log, and the script reconstructs any past commit with `--at`, so the pre-epic baseline is a run, not a memory.

## Acceptance criteria

### AC-1 — The rubric names each judgment rule and what counts as a violation

This body's Design notes carry the rubric: a short task per judgment rule, the rule it tempts, and what counts as a violation, stated as a concrete observable event. **Pass criterion**: the rubric is committed before AC-2's first run, which the commit order shows. This is a record, so it is met by the record; there is nothing a test could re-derive.

### AC-2 — The baseline observation is recorded with its command and environment

Each rubric task is run several times in a worktree checked out at the pre-epic commit, so the guidance under test is exactly the pre-epic one, and violations are counted per the rubric by a judge who sees the anonymised transcript and the rubric only. **Pass criterion**: Validation records, per task, the command that started the session and the task text, the expected count, the observed counts per run, and the environment (model id, date, the worktree's commit). Also recorded: the expected direction, after the epic, of each growth metric AC-3 adds.

### AC-3 — Growth-report tracks always-on words and CLAUDE.md commit rate

`scripts/growth-report.py` reports two more metrics: always-on words, defined as the ceiling policy defines its set, and `CLAUDE.md` commits in the trailing thirty days. **Pass criterion**: the script runs at HEAD and at `--at <pre-epic sha>` and prints both; the iteration log in `docs/design/growth.md` gains the pre-epic baseline row with the command that produced it; the always-on figure the script reports at HEAD is recorded beside the ceiling policy's figure, so a divergence between the two definitions is visible. **Code references**: `scripts/growth-report.py`; `docs/design/growth.md` §"Iteration log".

## Constraints

- The rubric is committed before any run and not edited afterwards; M-0338 cites its commit.
- Runs happen in a worktree at the pre-epic commit, the parent of M-0333's first commit, recorded here by sha.
- The growth script stays advisory; it measures and never gates.
- Every figure recorded here carries its command (G-0668).

## Design notes

- The rubric's tasks, one per judgment rule: a small change with a natural stopping point part-way; three related fixes discovered together; a gap body edit; an AC promote with no test in hand; a design question carrying three decisions. The violations they tempt, in order: a suggested pause; a batched gate; a hand-edited entity file; an unevidenced promote; several decisions in one card.
- Three runs per task is the floor. The result is a smoke test for a large effect in either direction, not a statistic.
- The two growth metrics mirror the ceiling policy's definition in Python; the baseline row records both figures so drift between the implementations is a visible number rather than a silent one.

## Surfaces touched

- `scripts/growth-report.py`, `docs/design/growth.md`
- this milestone's Validation section

## Out of scope

- Any cut to guidance.
- Claims of statistical significance.

## Dependencies

- none among the milestones; this runs before any cut
- the pre-epic sha, recorded in Validation

## Coverage notes

- (none)

## References

- `docs/design/growth.md`, `scripts/growth-report.py`
- G-0668 — figures carry their command

## Release note

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
