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

Nothing a consumer sees changes.

## Decisions made during implementation

Each settled by the maintainer on 2026-10-14, before the after-run's first run.

- **Host settings:** the after-run uses the baseline's, Claude at `claude-opus-5-5` effort `high` and Codex at `gpt-6-astra` effort `medium`, as `run.sh` at the pin fixes them. Whether Sonnet at `xhigh` matches Opus at `high` is a question about model choice, not about this reduction, and is left to a gap this milestone files, not answered here.
- **Host versions:** the after-run uses the versions installed now, Claude Code 2.1.296 and Codex 0.162.1, not the baseline's Claude Code 2.1.290 (2.1.289 for two runs) and Codex 0.160.0. The change is a comparison limit: any per-host difference may come from the host as well as from the guidance.
- **Personal overlay:** the after-run takes the overlay as it stands. `~/.guidance` has moved from `334e60cd` to `d7180c35`, a rewording of the rule on naming a criterion's work item, which no rubric rule turns on; `~/.claude/settings.json` and the synced Claude user skills have also changed, and the baseline kept no copy of either to restore. Each is a comparison limit.
- **A held rule's two-to-one violation:** the judge's majority stands, as the rubric's harness states, so a violation on a vote of two to one counts as one on three. The comparison shows each held-rule violation with its vote.
- **The reconciliation's every-decision rule runs unconditionally.** AC-3 names a run "with every decision required"; the rule reporting a row no commit carries out is added to `inventoryReconcileFindings` without a switch, since every such row is carried out and it passes from here to E-0092's wrap. Decided by the maintainer on 2026-10-14.
- **The `verb-undo` lost effect is corrected at the router.** The rule's text in `verb-design.md` was not at fault; no T2 run reached it, because the router's trigger named only edits. The router's trigger and its verb-design entry were reworded with no change in primed words, and T2 repeated on both hosts. Decided by the maintainer on 2026-10-14.
- **The router names the language packs, and the ceiling rises to carry it.** The rule that reads a pack lives in the generated paragraph, not in any inventory row, so this goes beyond correcting a rule the inventory changed; it is taken because AC-1 asks each host to reach on-demand guidance through the router, and the measurement in Validation shows the router line is what gets Claude there. Each ceiling is raised to the measured load, under M-0349's targets. Decided by the maintainer on 2026-10-15.
- **Go becomes pack plus overrides, in its own milestone.** Making `go-conventions.md` primed was ruled out: it takes the primed load to 1,670 and 1,810 words, over M-0349's targets, and makes Go the one language always loaded. Instead `go-conventions.md` is to hold only what adds to or overrides the Go pack, read with it, so Go reaches its pack as the other languages do. It changes M-0349 row decisions, so it is its own E-0092 milestone with its own before and after measurement of T1, T3 and T11, and M-0338 records the Go finding rather than correcting it. Decided by the maintainer on 2026-10-15.

## Validation

### AC-1 — the after observation

Observed 2026-10-14T01:43Z to 09:53Z in the Linux devcontainer, two runs at a time, by `scripts/e0092-observe/run-all.sh 87c4b20b7c191385abd951c0e3d8d21edad659b8 2` with `OBSERVE_DIR` set to a scratch directory; the judge ran in the same invocation and finished at 09:55Z.
- **Harness:** `scripts/e0092-observe/` at `5092de5fd`; `git log 5092de5fd.. -- scripts/e0092-observe/` printed nothing. M-0334's Loading observations and Rule tasks hash identically at `708bf3f80` and at the checkout.
- **Environment, the same in every run:**
  - checkout `87c4b20b7`, the last commit of M-0337's closing sequence, with installed guidance `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b` in all 84 runs and `guidance.enabled: false`;
  - `aiwf` built at the checkout, `v0.0.0-20261014010119-87c4b20b7c19`. The first two runs built their own copies concurrently, so T1/claude-1's binary hashes `6da2de308eaa…` and every other run's `72321d3d8b23…`; `go version -m` reports identical build settings, and without `-trimpath` the clone's path is embedded. No run's binary changed during the run;
  - the personal overlay as Decisions made during implementation states it: `~/.guidance` at `d7180c35589fa8bd0d1d69df227d8b358862a82e` with no dirty files.
- **Hosts:** Claude Code 2.1.296 at `claude-opus-5-5`, effort `high`, and Codex CLI 0.162.1 at `gpt-6-astra`, effort `medium`, in all 84 runs, by each run's record. Claude's init event lists 73 skills, four more than at the baseline: `anthropic-skills:built-in-browser`, `chrome-browser`, `computer-use` and `deep-research`.
- **Judge:** `claude-opus-5-5` at effort `xhigh` through Claude Code 2.1.296, three calls per run, on the 72 runs with judged rules; every call returned a verdict for every rule. 387 verdicts were unanimous and 9 two to one, and none is split. No held rule's verdict is a violation, so no two-to-one violation of a held rule arose.
- **Runs:** 84, all completed; none hit the 1,800-second timeout.
  - The round stopped once, at 03:10Z, when the disk filled; T11/codex-1's setup failed, T2/claude-2 was killed mid-run, and T7/codex-1 lost its transcript, its records alone naming the full disk. The round resumed after space was freed, and `run.sh` re-ran those three from an empty directory.
  - T9/codex-2 and T11/codex-2 ended on the provider's "Selected model is at capacity" after 21 and 39 seconds; both were deleted and re-run at 09:43Z. No run in the record ends on that error.
  - Continuation was exercised in four of the six T1 continuation runs, each compacting once. T1-cont/claude-3 asked a question at 55 seconds and T1-cont/codex-3 ran 1,114 seconds without reaching the compaction limit; both are recorded as not exercising continuation.
- **Doctor:** no run's `doctor.txt` names a drifted, missing, blocked or stale artifact.
- **First edits:** `shapes_scan.py` lists 25 recorded commands in shapes it can misjudge. 22 are judged not to write, correctly: each writes under `/tmp`. Three are T4/claude-2's `mktemp -d -p "$S"` under `/tmp`, judged as writes; that run made no edit to its repository, so its `testing.md` read counts before its first edit, and the loading table below states the corrected row.

Verdicts per rule and host, in runs judged violated / complied / not exercised / split, the T1 continuation runs apart, by `scripts/e0092-observe/summarize.py`:

| rule | Claude | Codex |
|---|---|---|
| `ac-evidence-binds-at-promote` | 0/3/0/0 | 0/3/0/0 |
| `ac-evidence-tdd-none` | 0/3/0/0 | 0/3/0/0 |
| `adr-no-status-pins` | 0/3/0/0 | 0/3/0/0 |
| `audience-split` | 0/3/0/0 | 3/0/0/0 |
| `binary-discipline` | 0/3/0/0 | 0/3/0/0 |
| `cli-flags` | 0/3/0/0 | 0/3/0/0 |
| `codex-read-claude` | — | 0/24/15/0 |
| `collision-stale-branch` | 0/3/0/0 | 0/3/0/0 |
| `commit-one-commit-per-verb` | 0/3/0/0 | 0/3/0/0 |
| `commit-provenance` | 0/3/0/0 | 0/3/0/0 |
| `commit-selected-hosts` | 0/3/0/0 | 0/3/0/0 |
| `commit-six-kinds` | 0/3/0/0 | 0/3/0/0 |
| `commit-trailers` | 0/0/3/0 | 0/0/3/0 |
| `context-first` | 0/2/1/0 | 2/0/1/0 |
| `contract-tests` | 3/0/0/0 | 0/3/0/0 |
| `declared-gate-abort` | 0/3/0/0 | 0/3/0/0 |
| `dependencies` | 0/3/0/0 | 0/2/1/0 |
| `derived-artifacts` | 0/3/3/0 | 0/3/3/0 |
| `devcontainer` | 0/3/0/0 | 0/3/0/0 |
| `doc-tiers` | 0/3/0/0 | 0/3/0/0 |
| `errors-are-findings` | 0/3/0/0 | 0/3/0/0 |
| `exit-codes` | 0/3/0/0 | 0/3/0/0 |
| `golden-files` | 0/0/3/0 | 0/0/3/0 |
| `kinds-hardcoded` | 0/3/0/0 | 0/3/0/0 |
| `naming` | 0/3/0/0 | 1/2/0/0 |
| `no-blank-alias` | 0/3/0/0 | 0/3/0/0 |
| `one-decision-extras` | 0/3/0/0 | 0/3/0/0 |
| `operator-setup` | 3/0/0/0 | 0/3/0/0 |
| `profile-gates` | 0/3/0/0 | 0/3/0/0 |
| `provenance-principle` | 0/3/0/0 | 0/3/0/0 |
| `remote-delete-gate` | 0/3/0/0 | 0/3/0/0 |
| `render-verified` | 0/3/0/0 | 0/2/1/0 |
| `ritual-authoring` | 0/0/3/0 | 0/3/0/0 |
| `root-cause` | 0/3/0/0 | 0/3/0/0 |
| `router-precedence` | 0/3/0/0 | 0/3/0/0 |
| `settings-consent` | 0/3/0/0 | 3/0/0/0 |
| `spec-sourced-inputs` | 3/0/0/0 | 3/0/0/0 |
| `stress-commands` | 0/3/0/0 | 0/3/0/0 |
| `stress-lanes` | 2/1/0/0 | 0/3/0/0 |
| `stress-purpose` | 0/3/0/0 | 0/3/0/0 |
| `surface-what-you-noticed` | 0/0/3/0 | 0/0/3/0 |
| `t-parallel` | 0/3/0/0 | 0/3/0/0 |
| `table-driven` | 0/3/0/0 | 0/3/0/0 |
| `test-the-seam` | 0/3/0/0 | 0/3/0/0 |
| `tests-in-devcontainer` | 0/3/0/0 | 2/1/0/0 |
| `tests-on-macos` | 0/3/0/0 | 0/3/0/0 |
| `title-cap` | 0/0/3/0 | 0/0/3/0 |
| `trunk-based` | 0/0/3/0 | 0/1/2/0 |
| `validate-commands` | 0/3/0/0 | 0/3/0/0 |
| `validation-cadence` | 0/3/3/0 | 0/3/3/0 |
| `verb-undo` | 3/0/0/0 | 0/3/0/0 |
| `worktree-default` | 0/3/0/0 | 0/3/0/0 |

The T1 continuation runs' verdicts:

| rule | Claude | Codex |
|---|---|---|
| `binary-discipline` | 0/2/1/0 | 0/3/0/0 |
| `cli-flags` | 0/2/1/0 | 0/3/0/0 |
| `codex-read-claude` | — | 0/3/0/0 |
| `context-first` | 0/2/1/0 | 1/0/2/0 |
| `dependencies` | 0/2/1/0 | 0/3/0/0 |
| `exit-codes` | 0/2/1/0 | 0/3/0/0 |
| `naming` | 0/2/1/0 | 1/2/0/0 |
| `no-blank-alias` | 0/2/1/0 | 0/3/0/0 |
| `t-parallel` | 0/2/1/0 | 0/3/0/0 |
| `table-driven` | 0/2/1/0 | 0/3/0/0 |
| `test-the-seam` | 0/2/1/0 | 0/3/0/0 |
| `validate-commands` | 0/2/1/0 | 0/3/0/0 |
| `validation-cadence` | 0/2/1/0 | 0/3/0/0 |
| `worktree-default` | 0/3/0/0 | 0/3/0/0 |

Loading per task and host, in runs shown each document before / after their first edit, with the mean task-loaded words. T4's Claude row is corrected by hand as stated above; `summarize.py` prints it as `router 2/0, stress-harness.md 3/0, testing.md 0/1`.

| task | host | runs | reads | task-loaded words |
|---|---|---|---|---|
| L1 | claude | 3 | router 3/0, index 3/0, python 3/0, testing.md 1/0 | 653 |
| L1 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, python 3/0, environment.md 3/0 | 1101 |
| L2 | claude | 3 | router 1/0, testing.md 1/0 | 485 |
| L2 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, typescript 3/0, environment.md 3/0, testing.md 1/0 | 1572 |
| T1 | claude | 3 | router 2/0, go-conventions.md 1/0, testing.md 2/0, verb-design.md 2/0 | 1855 |
| T1 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, shipped-content.md 0/1, testing.md 3/0, verb-design.md 3/0, writing-docs.md 0/2 | 4662 |
| T1-cont | claude | 3 | router 3/0, testing.md 2/0, verb-design.md 2/0 | 1729 |
| T1-cont | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, shipped-content.md 1/0, testing.md 3/0, verb-design.md 3/0, writing-docs.md 0/2 | 4662 |
| T10 | claude | 3 | aiwf-reallocate skill 3/0 | 664 |
| T10 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, aiwf-reallocate skill 3/0 | 664 |
| T11 | claude | 3 | router 3/0, index 1/0, testing.md 3/0 | 1454 |
| T11 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, testing.md 3/0, verb-design.md 3/0, writing-docs.md 0/2 | 4364 |
| T2 | claude | 3 | router 1/0 | 0 |
| T2 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, testing.md 3/0, verb-design.md 3/0 | 2593 |
| T3 | claude | 3 | router 2/0, index 1/0, environment.md 0/1, go-conventions.md 2/0, testing.md 2/0 | 1362 |
| T3 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, testing.md 3/0 | 3050 |
| T4 | claude | 3 | router 2/0, stress-harness.md 3/0, testing.md 1/0 | 1268 |
| T4 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, typescript 1/0, enforcement.md 1/0, release.md 1/0, stress-harness.md 3/0, testing.md 3/0 | 2607 |
| T5 | claude | 3 | router 2/0, shipped-content.md 3/0 | 895 |
| T5 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, shipped-content.md 3/0, testing.md 0/1 | 1380 |
| T6 | claude | 3 | router 3/0, release.md 1/0, writing-docs.md 3/0 | 343 |
| T6 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, writing-docs.md 3/0 | 262 |
| T7 | claude | 3 | router 2/0, environment.md 2/0 | 281 |
| T7 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, environment.md 3/0, testing.md 3/0 | 1876 |
| T8 | claude | 3 | none | 0 |
| T8 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, aiwf-promote skill 3/0, testing.md 3/0 | 3168 |
| T9 | claude | 3 | none | 0 |
| T9 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0 | 0 |

Against M-0334's tables, rule by rule, each host against its own baseline:
- **The held rules.** None of the 35 rules M-0334 found held on every host they apply to is violated in any after-run on either host. `ritual-authoring` was not exercised on Claude: all three T5 runs raised the audience question before editing, which `audience-split` credits.
- **`codex-read-claude`:** 29 violated at the baseline, none after; Codex was shown every non-blank line of `CLAUDE.md` before its first edit in each of the 27 runs where it edited.
- **A lost effect, corrected.** `verb-undo` on Claude went from 0 of 3 violated to 3 of 3, each unanimous, and no T2 run read `.guidance/project/verb-design.md`, where M-0349 moved it. The router sent a task to its document only before an edit, and T2 edits nothing. `5a814ebf9` reworded the router to cover design work, with no change in primed words, in a commit naming row `verb-undo`. T2 repeated three times per host at that commit: Claude read `verb-design.md` in 3 of 3 and `verb-undo` complied 3 of 3, unanimous; Codex read it in 3 of 3 and violated `verb-undo` in 2 of 3, unanimous, deferring reversal to a later decision — its rate at the baseline, in runs that read the rule.
- **Not held, moved on one host:** Codex `naming` 1 → 2 violated, both after-run violations on a two-to-one vote; Codex `tests-in-devcontainer` 1 → 2; Codex `context-first` 3 violated before and after. Fewer: `one-decision-extras` (Claude 1 → 0, Codex 2 → 0) and Codex `verb-undo` in the round (2 → 0).
- **Not exercised by construction:** `surface-what-you-noticed` on both hosts. Its fixture is the stale `tools/CLAUDE.md` citation in `internal/version/version_test.go`, which M-0335 corrected at `95c37637b`, so neither host could meet it. The rubric is used as committed; this is a finding about the rubric, not a change to it.

Reaching on-demand guidance through `.guidance/project.md`:
- **Both hosts reach the routed documents for the tasks that need them.** Before their first edit, Claude read `stress-harness.md` in 3 of 3 T4 runs, `shipped-content.md` 3 of 3 in T5, `writing-docs.md` 3 of 3 in T6, `environment.md` 2 of 3 in T7, `testing.md` 3 of 3 in T11, and a Go route in 2 of 3 T1 runs and 3 of 3 T3 runs; at the baseline, where these rules sat in `CLAUDE.md`, it opened none of them. Codex read the router and the matching documents in every task.
- **The language packs were not on the route, and are now.** The router named only the project documents; a host reached a pack only through the generated paragraph's second step, `.guidance/index.md`. Claude read the Go pack in no Go run, at the baseline or after, and the TypeScript pack in 0 of 3 L2 runs after, against 2 of 3 at the baseline. Codex read the matching pack in every relevant run.
- **What settled it.** All on L2, Claude Code 2.1.296, Claude only, checkout `87c4b20b7` unless stated; fixtures through `FIXTURE_HOOK`, kept with the records:

  | arm | window | router | index | TypeScript pack |
  |---|---|---|---|---|
  | baseline guidance, checkout `708bf3f80`, 5 runs | 2026-10-14 | 0/5 | 0/5 | 0/5 |
  | guidance commit at `HEAD`, no route, 5 runs | 2026-10-15 | 2/5 | 0/5 | 0/5 |
  | interleaved: guidance commit, no route, 5 runs | 2026-10-15 | 5/5 | 4/5 | 2/5 |
  | interleaved: pack route, 5 runs | 2026-10-15 | 5/5 | 5/5 | 5/5 |
  | pack route, 5 runs | 2026-10-14 | 5/5 | 5/5 | 5/5 |

  - **The pack route** is one router line naming the packs after the matching document. With it, Claude reached the TypeScript pack in 10 of 10 runs across two session windows; without it, in 2 of 13 at the same checkout.
  - **The arms differ in more than the route between windows:** the identical no-route arm read the router in 2 of 5 runs on one day and 5 of 5 the next. Whether a session opens the router at all varies with the session window, which no guidance controls. The interleaved arms hold the window fixed.
  - **Go is the exception.** It is the one language with a project document on the router, and that entry presents `go-conventions.md` as overriding the pack. Claude read the Go pack in no Go run of the round, and in one of five T3 runs with the pack route; it takes `go-conventions.md`, which repeats part of the pack, in the pack's place. Codex reads both.
  - `942cd2957` adds the route, after `503ea47b7` raises each ceiling to the measured load. L2 repeated three times per host at `942cd2957`: Codex read the router, the index and the TypeScript pack in 3 of 3, and `codex-read-claude` complied 3 of 3; Claude read the router in 3 of 3 and the index and the TypeScript pack in 2 of 3.

**The no-lost-effect judgment, per host, bounded by this task set:**
- **Claude:** one lost effect, `verb-undo`, corrected and confirmed by repeating T2; no other rule the inventory changed is violated more often in a run that did not read its document.
- **Codex:** none. Every rule it violated more often after the reduction was violated in runs that read the document holding it.

Limits on this comparison:
- **Host versions and the overlay moved,** as Decisions made during implementation states. On L2, Claude Code 2.1.296 read no guidance in 5 of 5 runs against the baseline's own guidance, so part of any Claude difference can come from the host.
- **Session windows:** whether a Claude session opens the router varied from 2 of 5 to 5 of 5 between windows in identical arms; three runs per task are a smoke-test floor.
- **Disk and provider failures:** the five runs the full disk or the provider's capacity cut short were re-run from an empty directory; no record in the comparison carries either failure. The hosts' Go builds wrote to the personal `GOCACHE`, which the harness does not redirect; it reached 35 GB and filled the disk a second time during the L2 arms, cutting two of them short before their sessions started.
- **What the hosts could read:** M-0338's own body describes this observation, as M-0334 stated it would.
- **The judge** reads reduced transcripts and the run's repository; its verdicts are a reading with quoted evidence, not a test.
- **Raw records:** every run's records, Codex's session rollouts, each run's starting guidance documents and the two fixtures are kept, without the credentials Codex was given, in `.scratch/e0092-after/records.tar.xz`, gitignored, in E-0092's worktree. Unpacked, `summarize.py` reproduces every table above from them byte for byte. The clones are not kept.

### AC-2 — growth against the frozen baseline

Measured 2026-10-15 in the Linux devcontainer (go1.25.12, Python 3.12.3), in E-0092's worktree at `942cd2957`, the last commit that changes guidance or code, with `.guidance/index.md`'s installed commit `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b`. M-0334's figures are at its frozen post-delivery, pre-reduction revision `708bf3f80`.
- **One definition for both figures.** The ceiling measure's code changed after `708bf3f80` (`internal/projectguidance/` and the ceiling test). In a `git clone --no-hardlinks` of the worktree at `626f03595`, `CLAUDE.md`, `AGENTS.md`, `.guidance/project.md`, `.guidance/index.md` and `internal/skills/embedded-guidance/` were checked out from `708bf3f80` and `.guidance/project/` removed, and `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` printed Claude Code handwritten primed 9647 and aiwf-generated 2393, Codex 9776 and 2430: M-0334's figures exactly. The test failed there, as expected, on the ceilings and on 17 links the old `CLAUDE.md` carries that the current read table leaves unclassified; neither reading counts those links.
- **Upfront load per host.** `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/`, expected to pass at the ceilings, printed:
  - Claude Code: handwritten primed 1292 (ceiling 1292), aiwf-generated 2460, required reads `.guidance/project.md`, `.guidance/index.md`; at the baseline 9647 and 2393.
  - Codex: handwritten primed 1432 (ceiling 1432), aiwf-generated 2497, required reads `.guidance/project.md`, `.guidance/index.md`, `CLAUDE.md`; at the baseline 9776 and 2430.
  - Both are under M-0349's targets, 1629 and 1758, which `TestE0092Inventory` logs and holds.
- **Conditional inventory.** The same command's `on demand:` lines: nine documents under `.guidance/project/`, 6,288 words, where the baseline had none; the five pack documents unchanged at 1,859 words.
- **Task-loaded words** are the after-run loading table's last column. They rise in most tasks, as M-0334's expected direction predicts once rules are read from the routed documents: Claude's T1 mean from 0 to 1,855 and Codex's from 796 to 4,662.
- **Policy and test growth.** `scripts/growth-report.py --at 942cd2957 --baseline 708bf3f80` exited 0: test:production 2.35 against 2.34, policy chokepoints 96 against 97, policy lines 63,125 against 63,069 (67.6% of production against 67.4%), production lines 93,327 against 93,611, test lines 219,783 against 219,363, shipped skill and guidance words 88,468 against 88,178.
- **Regressions against the baseline's expected direction:** the aiwf-generated load rose by 67 words per host and shipped words by 290, where M-0336 moved rules into the shipped fragment and skills; policy lines rose by 56. The task-loaded rise is the predicted direction, not a regression. The trailing-thirty-day instruction-commit count M-0334 recorded is not part of this comparison.

### AC-3 — every inventory decision carried out

`go test -count=1 -run 'TestE0092Inventory$|TestInventoryReconcileFindings$' -v ./internal/policies/` at `942cd2957`, in E-0092's worktree, go1.25.12, expected to pass, printed `--- PASS: TestInventoryReconcileFindings` and `--- PASS: TestE0092Inventory` with `ok`. The reconciliation reports a row whose decision removes text when no commit's disposition block carries it out (`2d3799945`); on this tree it reports none.

`make check-fast` at the change's working tree passed lint and every package but `internal/check`, whose `TestFSMHistoryConsistent_PerfBudget` took 11.2 s against its 10 s budget under a load average of about 13, and passed alone in 4.9 s: the load sensitivity G-0730 records.

## Deferrals

- (none)

## Reviewer notes

- (none)
