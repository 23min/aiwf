---
id: M-0338
title: Record the after observation and re-run the growth report
status: in_progress
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
      status: met
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

Run the ceiling measure and the growth report at HEAD, and compare them with M-0334's figures from its frozen post-delivery, pre-reduction revision. Where the ceiling measure's code has changed since that revision, re-measure that revision's guidance with the current code, so both figures come from one definition, and record how. Append the commands and results to the growth iteration log. Compare upfront counts per host, conditional inventory, observed task-loaded text and policy/test growth against the baseline expectations; name regressions. Do not use the pre-delivery snapshot as this comparison's baseline.

### AC-3 — Every inventory decision has been carried out

Every row of M-0349's inventory whose decision changes text has a commit that carried it out and names the row in its disposition block. **Pass criterion**: the inventory test's reconciliation (M-0336 AC-5), run with every decision required, reports a row with no carrying commit; on the tree at this milestone's end it reports none. Validation records that last run, since E-0092's wrap deletes the test after it. A row left undone is carried out, or changed by a maintainer decision recorded in M-0349, before the epic wraps.

## Constraints

- No cut lands between M-0337's last commit and the after-run.
- The rubric is used as committed; a needed change is recorded as a finding here, not applied.
- Every figure carries its command (G-0668).

## Design notes

- State the no-lost-effect judgment separately for each host; it is bounded by the recorded task set, not a universal compliance guarantee.
- A changed rule the after-check finds not followed is corrected at its home, in a guidance commit that names the rule's inventory row, and the affected task is repeated before the epic wraps.
- Before the after-run, settle which host settings to observe. M-0334 ran Claude at `claude-opus-5-5` with effort `high`, while the maintainer's own sessions run at `xhigh` and the builder agent runs on Sonnet, which neither round covers. Whether Sonnet at `xhigh` matches or beats Opus at `high` on cost, tokens or quality is open. Settings that change between rounds void the comparison, so a change means re-observing M-0334's baseline under the new settings first.

## Surfaces touched

- this milestone's Validation section
- the home of any rule the after-check finds not followed
- `docs/design/growth.md` §"Iteration log"

## Out of scope

- Any change to guidance beyond correcting a rule the after-check finds not followed.

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

Each settled by the maintainer on 2026-10-14, before the after-run's first run.

- **Host settings:** the after-run uses the baseline's, Claude at `claude-opus-5-5` effort `high` and Codex at `gpt-6-astra` effort `medium`, as `run.sh` at the pin fixes them. Whether Sonnet at `xhigh` matches Opus at `high` is a question about model choice, not about this reduction, and is left to a gap this milestone files, not answered here.
- **Host versions:** the after-run uses the versions installed now, Claude Code 2.1.296 and Codex 0.162.1, not the baseline's Claude Code 2.1.290 (2.1.289 for two runs) and Codex 0.160.0. The change is a comparison limit: any per-host difference may come from the host as well as from the guidance.
- **Personal overlay:** the after-run takes the overlay as it stands. `~/.guidance` has moved from `334e60cd` to `d7180c35`, a rewording of the rule on naming a criterion's work item, which no rubric rule turns on; `~/.claude/settings.json` and the synced Claude user skills have also changed, and the baseline kept no copy of either to restore. Each is a comparison limit.
- **A held rule's two-to-one violation:** the judge's majority stands, as the rubric's harness states, so a violation on a vote of two to one counts as one on three. The comparison shows each held-rule violation with its vote.

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
