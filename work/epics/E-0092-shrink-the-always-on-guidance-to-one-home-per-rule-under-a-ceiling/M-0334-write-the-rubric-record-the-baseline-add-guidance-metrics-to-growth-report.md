---
id: M-0334
title: Write the rubric, record the baseline, add guidance metrics to growth-report
status: draft
parent: E-0092
depends_on:
    - M-0350
tdd: none
acs:
    - id: AC-1
      title: The rubric names each judgment rule and what counts as a violation
      status: open
    - id: AC-2
      title: The baseline observation is recorded with its command and environment
      status: open
    - id: AC-3
      title: Growth-report measures upfront guidance per host and guidance commit rate
      status: open
---

## Goal

Freeze the installed guidance and record the post-delivery, pre-reduction baseline: how Claude and Codex load and follow guidance on the same bounded task set.

## Closes

- (none)

## Context

E-0094 delivered the project guidance and migrated this repository. M-0348 recorded root-started sessions at its migration commit `9f817fa5b34cdcc4e0a065f7f7e78a54a030ad83`: Codex on a nested Go task, and Claude on a nested Go task, a Python-script task, a new-file task and a prose-only task. The guidance has changed since — `AGENTS.md` gained the rendered aiwf fragment and `CLAUDE.md` gained rules — so those sessions are not this baseline. Their harness — exact command lines, expectations written before each run, reads judged on tool events — is the harness here, and their findings shape the expected directions below. M-0338 compares against this milestone's committed rubric, source revision and checkout. The historical growth report remains useful, but it cannot reconstruct machine-local instructions or prove a host read a file; capture those observations explicitly.

## Acceptance criteria

### AC-1 — The rubric names each judgment rule and what counts as a violation

This body's Design notes carry the rubric: a short task per judgment rule, the rule it tempts, and what counts as a violation, stated as a concrete observable event. **Pass criterion**: the rubric is committed before AC-2's first run, which the commit order shows. This is a record, so it is met by the record; there is nothing a test could re-derive.

### AC-2 — The baseline observation is recorded with its command and environment

Run each rubric task for both hosts in worktrees at the same post-delivery, pre-reduction commit. Record command/prompt, expectation, observed reads and behavior, host/model versions, date, checkout commit, guidance source revision and personal overlay. A judge receives the rubric and anonymised transcript. Record observed violations and unavailable evidence without treating file presence as a read. Freeze the installed guidance during the comparison and state the expected direction of each metric.

### AC-3 — Growth-report measures upfront guidance per host and guidance commit rate

Extend the report with two figures per host — handwritten primed words and the rendered fragment's words — plus inventory sizes for on-demand guidance, and trailing-thirty-day commits affecting either host's instructions. Use the same source-set definition as M-0333 and compare its counts against the policy. Report personal/global words and observed task-loaded words in the baseline record rather than claiming Git can reconstruct them. **Pass criterion**: run the report at HEAD and at the frozen baseline, test the metrics on fixtures, and append a dated baseline row with commands to the growth document.

## Constraints

- Set `guidance.enabled: false` in `aiwf.yaml` before the first baseline run, so no `aiwf update` or `aiwf worktree add` refreshes the installed packs; it stays off until M-0338 completes. Confirm `.guidance/index.md`'s installed commit at every observation.
- Commit the rubric before its first run and do not change it during comparison.
- Record the post-delivery baseline commit before any reduction; use it for both hosts.
- Keep host/model settings and personal overlays fixed between paired runs; disclose unavoidable changes as comparison limits.
- No upstream refresh between before and after observations.
- Growth reporting stays advisory. Every figure carries its command and environment.

## Design notes

- Judgment tasks cover continuation through a bounded task, approval for related local changes versus outward actions, a gap body edit, an AC promotion with inadequate evidence, and a design question with several decisions.
- A declared, enumerated local reversible approval sequence is valid. A batch containing an outward action is a violation. Editing entity body prose for review is valid; hand-editing frontmatter or committing without the verb is not.
- Guidance-loading tasks cover Go implementation, a Python script, TypeScript tests, prose-only work, and creating a new file. Start at repository root and observe whether relevant guidance is read before the first relevant edit; prose-only work should not require full language documents. Neither host has been observed on the TypeScript task, and Codex only on the nested Go task.
- Record the expected direction after the reduction for each loading task. M-0348 observed Claude answering from `CLAUDE.md` without opening the router whenever `CLAUDE.md` held the answer, and never reading the code-health primer; once the root is thinned, the prediction is that it follows the router.
- Repeat the task set for both hosts, including a continuation after compaction. Three runs per task is a smoke-test floor, not evidence of statistical significance.
- Record installed files, observed reads and behavioral compliance as separate observations.
- Runs that invoke `aiwf update` or `aiwf worktree add` point `HOME` and the XDG directories at a scratch directory, so the personal statusline and settings are untouched.

## Surfaces touched

- `scripts/growth-report.py`, `docs/design/growth.md`
- this milestone's Validation section

## Out of scope

- Any cut to guidance.
- Claims of statistical significance.

## Dependencies

- E-0094 — done; M-0348 holds the harness this baseline reuses.
- M-0333's source-set definition must agree with the reported measure before the baseline is accepted.
- The frozen checkout and guidance revisions, recorded in Validation.

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
