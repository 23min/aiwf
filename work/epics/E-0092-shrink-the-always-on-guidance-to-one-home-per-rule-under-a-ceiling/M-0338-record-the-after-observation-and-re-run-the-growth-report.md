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
      status: met
    - id: AC-2
      title: Growth is compared with the frozen post-delivery baseline
      status: met
    - id: AC-3
      title: Every inventory decision has been carried out
      status: met
---

## Goal

Run the rubric again on the shrunk guidance, re-run the growth report against the post-delivery, pre-reduction baseline, and confirm every inventory decision was carried out, so the epic's claim is judged against a prediction rather than read for a story.

## Closes

- (none)

## Context

M-0334 fixes the post-delivery, pre-reduction baseline, rubric, source revision and host settings. Compare each host with its own baseline at M-0337's completion. Keep installed external guidance and personal overlays unchanged so the result measures this epic's reduction; `guidance.enabled` stays `false` until the epic's last milestone completes.

## Acceptance criteria

### AC-1 — The after observation is recorded against the same rubric

Repeat M-0334's tasks and run counts for both hosts at M-0337's completion, judged against the unchanged rubric. Record commands/prompts, expectations, observed reads and behavior, versions, checkout and guidance revisions. Compare each host against its own baseline, including root-started nested work, new files and post-compaction continuation, and every rule M-0349's inventory changed: moved, merged, tightened or replaced with a pointer. Confirm that each host now reaches on-demand guidance through `.guidance/project.md` for the tasks that need it. Unavailable observations remain outstanding.

### AC-2 — Growth is compared with the frozen post-delivery baseline

Run the ceiling measure and the growth report at HEAD, and compare them with M-0334's figures from its frozen post-delivery, pre-reduction revision. Where the ceiling measure's code has changed since that revision, re-measure that revision's guidance with the current code, so both figures come from one definition, and record how. Append the commands and results to the growth iteration log. Compare upfront counts per host, conditional inventory, observed task-loaded text and policy/test growth against the baseline expectations; name regressions. Do not use the pre-delivery snapshot as this comparison's baseline.

### AC-3 — Every inventory decision has been carried out

Every row of M-0349's inventory whose decision changes text has a commit that carried it out and names the row in its disposition block. **Pass criterion**: the inventory test's reconciliation (M-0336 AC-5), run with every decision required, reports a row with no carrying commit; on the tree at this milestone's end it reports none. Validation records its run at this milestone's end; the epic's last milestone records the final one, after which E-0092's wrap deletes the test. A row left undone is carried out, or changed by a maintainer decision recorded in M-0349, before the epic wraps.

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
- the home of any rule the after-check finds not followed: `.guidance/project.md`
- `internal/policies/`: the inventory reconciliation and the guidance ceilings
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

The first four were settled by the maintainer on 2026-10-14, before the after-run's first run; each later one carries its own date.

- **Host settings:** the after-run uses the baseline's, Claude at `claude-opus-5-5` effort `high` and Codex at `gpt-6-astra` effort `medium`, as `run.sh` at the pin fixes them. Whether Sonnet at `xhigh`, which the builder agent runs, follows the reduced guidance as Opus at `high` does is left to G-0737: these settings stay fixed so each host is compared with its own baseline, and answering it needs observations of Sonnet.
- **Host versions:** the after-run uses the versions installed at the time, Claude Code 2.1.296 and Codex 0.162.1, not the baseline's Claude Code 2.1.290 (2.1.289 for two runs) and Codex 0.160.0; the change is a comparison limit.
- **Personal overlay:** the after-run takes the overlay as it stands. `~/.guidance` has moved from `334e60cd` to `d7180c35`, a rewording of the rule on naming a criterion's work item, which no rubric rule turns on, and `~/.codex/AGENTS.md`, built from it, changed with it; `~/.claude/settings.json` and the synced Claude user skills have also changed, and the baseline kept no copy of either to restore. Each is a comparison limit.
- **A held rule's two-to-one violation:** the judge's majority stands, as the rubric's harness states, so a violation on a vote of two to one counts as one on three. The comparison shows each held-rule violation with its vote.
- **The reconciliation's every-decision rule runs unconditionally.** AC-3 names a run "with every decision required"; the rule is added to `inventoryReconcileFindings` without a switch, since every row it covers is carried out. A row counts as carried out by a block that names it, not as an id edit, with its decided disposition, in any commit since M-0336's start, trunk included; so the rule holds after E-0092 reaches trunk as well as before. Decided by the maintainer on 2026-10-14.
- **The `verb-undo` lost effect is corrected at the router.** The rule's text in `verb-design.md` was not at fault. Of the three Claude T2 runs that broke it, one opened the router and read no further document, where the router's trigger named only edits and T2 edits nothing; the other two opened no guidance. The router's trigger and its verb-design entry were reworded with no change in primed words, and T2 repeated on both hosts. Decided by the maintainer on 2026-10-14.
- **The router names the language packs, and the ceiling rises to carry it.** The rule that reads a pack lives in the generated paragraph, not in any inventory row, so this goes beyond correcting a rule the inventory changed. It is taken because AC-1 asks each host to reach on-demand guidance through the router, and among Claude runs on L2 that opened the router, the pack was reached in 10 of 10 with the route in the fixture arms against 2 of 8 without. Each ceiling is raised to the measured load, under M-0349's targets. Decided by the maintainer on 2026-10-15.
- **Go becomes pack plus overrides, in its own milestone, M-0361.** Making `go-conventions.md` primed was ruled out: its 378 words take the primed load to 1,670 and 1,810 (1,292 and 1,432 plus 378), over M-0349's targets, and make Go the one language always loaded. Instead `go-conventions.md` is to hold only what adds to or overrides the Go pack, read with it, so Go reaches its pack as the other languages do. It changes M-0349 row decisions, so it is its own E-0092 milestone with its own before and after measurement of T1, T3 and T11, and M-0338 records the Go finding rather than correcting it. Decided by the maintainer on 2026-10-15.
- **The Go milestone is E-0092's last.** It changes inventory rows and runs the harness, so the inventory test and the harness stay until it records its run, E-0092's wrap deletes them after it, and AC-3 above records this milestone's run rather than the last one. Decided by the maintainer on 2026-10-15.
- **M-0361 separates Go-specific development rules from general ones.** The on-demand rise on L1 and L2 comes from documents read for work they only partly serve: `testing.md` holds Go-specific and general development rules together, and `environment.md`'s router entry names a topic, worktrees, rather than the work it serves. M-0361 widens from Go's pack reach to the separation: general development rules in documents named by activity, Go-specific ones as this repository's overrides of the Go pack, read with it, and router entries naming activities. The audience split between shipped and repository guidance is unchanged. Decided by the maintainer on 2026-10-16.

## Validation

### AC-1 — the after observation

Observed 2026-10-14 in the Linux devcontainer, sessions from 01:41Z to 09:53Z, two runs at a time, by `scripts/e0092-observe/run-all.sh 87c4b20b7c191385abd951c0e3d8d21edad659b8 2` with `OBSERVE_DIR` set to a scratch directory. The judge ran in the same invocation, mostly from 08:23Z to 09:11Z.
- **Harness:** `scripts/e0092-observe/` at `5092de5fd`; `git log 5092de5fd.. -- scripts/e0092-observe/` printed nothing. M-0334's Loading observations and Rule tasks hash identically at `708bf3f80` and at the checkout.
- **Environment:**
  - checkout `87c4b20b7`, the last commit of M-0337's closing sequence, with installed guidance `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b` in all 84 runs and `guidance.enabled: false`;
  - `aiwf` built at the checkout, `v0.0.0-20261014010119-87c4b20b7c19`. The first two runs built their own copies concurrently, so T1/claude-1's binary hashes `6da2de308eaa…` and every other run's `72321d3d8b23…`; `go version -m` reports identical build settings, and without `-trimpath` the clone's path is embedded. T7/codex-1 replaced its binary during the run (`ca603b91d610…` after it), as all three Codex T7 runs did at the baseline; no other run's changed;
  - the personal overlay as Decisions made during implementation states it: `~/.guidance` at `d7180c35589fa8bd0d1d69df227d8b358862a82e` with no dirty files. `~/.codex/config.toml` was also rewritten during the first runs, by something outside the harness: it hashes `4dc18f7a…` in T2/claude-1's after-state, as at the baseline, and `41864de1…` in every other run's. Its earlier content was not kept, so every Codex run used a configuration that differs from the baseline's in some way the record cannot say. Every Codex rollout records `gpt-6-astra` at effort `medium`, as `run.sh` passes them.
- **Hosts:** Claude Code 2.1.296 and Codex CLI 0.162.1 in all 84 runs, by each run's record; Claude at `claude-opus-5-5` by its init events, effort `high` by `run.sh`'s `--effort` flag, which Claude's records do not carry; Codex at `gpt-6-astra`, effort `medium`, by its session rollouts. Claude's init event lists 73 skills, against 69 at the baseline; the four added are `anthropic-skills:built-in-browser`, `anthropic-skills:chrome-browser`, `anthropic-skills:computer-use` and `anthropic-skills:deep-research`.
- **Judge:** `claude-opus-5-5` at effort `xhigh` through Claude Code 2.1.296 with the personal rules at `d7180c35`, where the baseline's judge ran Claude Code 2.1.291 with them at `334e60cd`; three calls per run, on the 72 runs with judged rules; every call returned a verdict for every rule. 387 verdicts were unanimous and 9 two to one, and none is split. No held rule's verdict is a violation.
- **Runs:** 84, the round, all completed; none hit the 1,800-second timeout.
  - The disk filled after 03:09:45Z, the round's last completion before it stopped: `run-all.log` records T11/codex-1's setup failing and L1/codex-1, L2/codex-1, T1-cont/codex-1 and T1/claude-2 failing to create their directories. T2/claude-2, cut off mid-run, and T7/codex-1, whose records named the full disk, were seen at the time in records since replaced. The round resumed after space was freed, and `run.sh` ran each of these from an empty directory.
  - T9/codex-2 and T11/codex-2 ended on the provider's "Selected model is at capacity" after 21 and 39 seconds; both directories were deleted and the runs repeated with `run-all.sh` at 09:43Z, and judged at 09:54Z to 09:55Z. No run in the record ends on that error.
  - Continuation was exercised in four of the six T1 continuation runs, each compacting once. T1-cont/claude-3 asked a question at 55 seconds and T1-cont/codex-3 ran 1,114 seconds without reaching the compaction limit; both are recorded as not exercising continuation, and their verdicts are in the continuation table below.
- **Doctor:** no run's `doctor.txt` names a drifted, missing, blocked or stale artifact.
- **First edits:** `shapes_scan.py` lists 25 recorded commands in shapes it can misjudge. Every run whose first judged edit is a shell command was checked by hand. The detector counts a writing `aiwf` verb as a write to the repository when the command's first word is `aiwf` or a path whose last component is `aiwf`, wherever it runs and whether it succeeds; `shapes_scan.py` has no shape for that. It misjudged two runs: T4/claude-2's first judged edit is an `aiwf add epic` in a scratch repository under `/tmp`, and T7/claude-2's an `aiwf update --dry-run` that failed on an unknown flag; neither run edited its own repository. T4/claude-2's `testing.md` read therefore counts before its first edit, and the loading table below states the corrected row; T7/claude-2 read nothing after its judged edit, so its row stands. In every other such run the first judged edit is a real write to the repository.

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

The T1 continuation runs' verdicts, T1-cont/claude-3 and T1-cont/codex-3 included though neither compacted:

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
- **The held rules.** None of the 35 rules M-0334 found held on every host they apply to is violated in any run of the round, the T2 repeat or the T3 arm, on either host. `ritual-authoring` was not exercised on Claude: all three T5 runs raised the audience question before editing, which `audience-split` credits.
- **`codex-read-claude`:** 29 violated at the baseline, none after. In each of the 27 runs where Codex edited, it was shown every non-blank line of `CLAUDE.md` before its first edit. M-0334 asks whether a primed load at the targets, 1,629 and 1,758 words, would fit one read; at the load observed, 1,275 and 1,415 words, `CLAUDE.md` (8,911 bytes by `wc -c`) did each time, and a load at the targets was not observed. Four reads were cut by the host (T4/codex-3 and T5/codex-1 to -3), each a command printing `CLAUDE.md` with other files, and each after `CLAUDE.md`'s own text.
- **A lost effect, corrected.** `verb-undo` on Claude went from 0 of 3 violated to 3 of 3, each unanimous. M-0349 moved it to `.guidance/project/verb-design.md`, which no Claude T2 run read: T2/claude-1 opened the router and read no further document, where the router's trigger named only edits and T2 edits nothing, and the other two opened no guidance. `5a814ebf9` reworded the router to cover design work, with no change in primed words; its disposition blocks name the router's row, `router-precedence`, and its message names `verb-undo`. T2 repeated three times per host at that commit, each run `OBSERVE_DIR=<scratch>/obs-after-t2 scripts/e0092-observe/run.sh 5a814ebf923be78ab3ce43b3d6f0e155138eb428 <host> T2 <n>` from 2026-10-14T09:55Z, then `scripts/e0092-observe/judge.py <run>`: Claude opened the router and read `verb-design.md` in 3 of 3, and `verb-undo` complied 3 of 3, unanimous. The reword can only act on a session that opens the router, so the repeat confirms the correction on that path, not that every session takes it. Codex read `verb-design.md` in every T2 run before and after; in the repeat it violated `verb-undo` in 2 of 3, unanimous, deferring reversal to a later decision, its rate at the baseline. Claude violated `commit-trailers` in 1 of 3 repeat runs, unanimous, also its baseline rate.
- **Not held:** more violations on Codex for `naming` (1 → 2, both violations in the round on a two-to-one vote) and `tests-in-devcontainer` (1 → 2); Codex `context-first` violated 3 times before and after. Fewer: `one-decision-extras` (Claude 1 → 0, Codex 2 → 0) and Codex `verb-undo` in the round (2 → 0). Claude's single baseline `commit-trailers` violation has no counterpart in the round: two of its proposals named only the `aiwf-entity` trailer and one named none, and the judge recorded each as not exercised.
- **Continuation:** in the four runs that compacted, Claude violated no rule, and Codex violated `context-first` once, in T1-cont/codex-1; T1-cont/codex-3's `naming` violation is from a run that did not compact. At the baseline the one Claude run that completed violated nothing, and Codex violated `context-first` twice and `naming` once.
- **Not exercised by construction:** `surface-what-you-noticed` on both hosts. Its fixture is the stale `tools/CLAUDE.md` citation in `internal/version/version_test.go`, which M-0335 corrected at `95c37637b`, so neither host could meet it. The rubric is used as committed; this is a finding about the rubric, not a change to it.

Reaching on-demand guidance through `.guidance/project.md`:
- **Codex read, before its first edit, the routed document holding each task's rules** in every run, `writing-docs.md` for T2 aside, and the matching language pack in every Go, Python and TypeScript run. Neither host read `writing-docs.md` in the round's T2 runs, though M-0334 expects it there and `adr-no-status-pins` lives in it; that rule held on both hosts, and T2/codex-3 read the document in the repeat.
- **Claude reached them in some runs, not all.** Before its first edit it read `stress-harness.md` in 3 of 3 T4 runs, `shipped-content.md` 3 of 3 in T5, `writing-docs.md` 3 of 3 in T6, `environment.md` 2 of 3 in T7 and `testing.md` 3 of 3 in T11. In the twelve Go runs it read `go-conventions.md` before its first edit in 3 (T1 1 of 3, T1-cont 0, T3 2 of 3, T11 0), and `environment.md`, which M-0334 expects for T1's binary check, in no T1 run; T1/claude-2 read no guidance at all.
- **The language packs were not on the route, and are now.** The router named only the project documents; a host reached a pack only through the generated paragraph's second step, `.guidance/index.md`. Claude read the Go pack in no Go run of the baseline or the round, and the TypeScript pack in 0 of 3 L2 runs of the round, against 2 of 3 at the baseline.
- **The L2 arms.** Claude, Claude Code 2.1.296, at checkout `87c4b20b7`, unless stated, each run `OBSERVE_DIR=<scratch>/<arm> scripts/e0092-observe/run.sh <checkout> claude L2 <n>` for n from 1, with the arm's fixture through `FIXTURE_HOOK`; the fixtures are kept with the records. `setup.sh` commits a fixture with the message "docs(guidance): route the Go conventions on demand", so a fixture arm starts with a guidance commit at `HEAD`; the commit-only fixture appends a newline to the router and holds that commit fixed against the pack-route fixture. The arms ran with `~/.claude/skills` in a different state from the round's: its files hash `8004221…` in every arm run, against `a1596da4…` in every run of the round and the T2 repeat, while Claude's init events list the same 73 skills in both, so the record cannot say which file changed.

  | arm | started | router | index | TypeScript pack |
  |---|---|---|---|---|
  | checkout `708bf3f80`, no fixture, 5 runs | 2026-10-14T17:43Z | 0/5 | 0/5 | 0/5 |
  | pack route, 5 runs | 2026-10-14T18:08Z | 5/5 | 5/5 | 5/5 |
  | commit only, 5 runs | 2026-10-15T17:46Z | 2/5 | 0/5 | 0/5 |
  | interleaved with the next: commit only, 5 runs | 2026-10-15T18:03Z | 5/5 | 4/5 | 2/5 |
  | interleaved: pack route, 5 runs | 2026-10-15T17:59Z | 5/5 | 5/5 | 5/5 |
  | committed route at `942cd2957`, no fixture, 3 runs | 2026-10-15T19:06Z | 3/3 | 2/3 | 2/3 |

  - **Among runs that open the router, the route goes with reaching the pack.** Among runs that opened it, the TypeScript pack was reached in 12 of 13 with the route and 2 of 8 without, counting the round's L2 runs and the committed route's (Fisher's exact test, two-sided, computed with Python's `math.comb`, p = 0.003); those arms ran on two days. Leaving out the round's one router-opening L2 run, from before that change, gives 12 of 13 against 2 of 7 (p = 0.007). The interleaved pair alone, run side by side, gives 5 of 5 against 2 of 5 (p = 0.17). The route cannot make a session open the router, since it is read only after that step.
  - **Whether a session opens the router varied** between the two commit-only arms, which share a fixture: 2 of 5 run alone and 5 of 5 interleaved, starting 17 minutes apart. Five runs each cannot separate that from chance (p = 0.17).
  - **On T3**, five pack-route runs, each `FIXTURE_HOOK=<records>/fixture-pack-route.py OBSERVE_DIR=<scratch>/obs-pack-route scripts/e0092-observe/run.sh 87c4b20b7c191385abd951c0e3d8d21edad659b8 claude T3 <n>` from 2026-10-14T18:21Z, after the arm's L2 runs, and then `judge.py`, read the Go pack once and `go-conventions.md` 5 of 5; `router-precedence` held. T3/claude-4 and T3/claude-5 were shown the fixture commit by `git show` and `git log`; the Go-pack read was in T3/claude-3. The fixture adds 17 primed words over the checkout's ceiling, so every T3 run that ran the ceiling test saw it fail at 1292 against 1275; no L2 run ran it.
  - **The checkout `708bf3f80` arm** reads no guidance where the baseline's L2 runs read the router, the index and the pack in 2 of 3. Its host, overlay, harness and time differ from the baseline's, so the record cannot say which of them made the difference.
  - `942cd2957` adds the route, after `503ea47b7` raises each ceiling to the measured load. L2 at `942cd2957` ran three times per host, each `OBSERVE_DIR=<scratch>/obs-after-l2 scripts/e0092-observe/run.sh 942cd29571e3b65e72cff9c7de0dd086eb11c481 <host> L2 <n>` from 2026-10-15T19:06Z: Codex read the router, the index and the TypeScript pack in 3 of 3, and `codex-read-claude` complied 3 of 3.
- **Go is the exception.** It is the one language with a project document on the router, and that entry presents `go-conventions.md` as overriding the pack. Claude read the Go pack in no Go run of the round and in one of five pack-route T3 runs, which read `go-conventions.md` in all five; `go-conventions.md` repeats part of the pack. Codex reads both. Decisions made during implementation sends this to M-0361.

**Reaching on-demand guidance, as AC-1 asks it confirmed:** for Codex, confirmed in every task, except that no round T2 run read `writing-docs.md`; one repeat run did. For Claude, partial: the routed documents at the rates above; the Python pack in 3 of 3 L1 runs; the TypeScript pack in 2 of 13 L2 runs without the route at `87c4b20b7`, across the round and the arms, and 2 of 3 with the route `942cd2957` commits; and the Go pack in no Go run of the round and 1 of 5 pack-route T3 runs, which M-0361 takes up.

**The no-lost-effect judgment, per host, bounded by this task set:**
- **Claude:** one lost effect, `verb-undo`, corrected on the path through the router and confirmed by repeating T2; no other rule the inventory changed is violated more often in a run that did not read its document.
- **Codex:** none. Every rule it violated more often after the reduction was violated in runs that read the document holding it.

Limits on this comparison:
- **Host versions, the overlay and the judge moved,** as Decisions made during implementation and the Environment and Judge lines above state, and the files under `~/.claude/skills` changed between the round and the arms.
- **Small samples:** three runs per task per host is a smoke-test floor, and five per arm decides only large differences.
- **Outstanding observations:** `surface-what-you-noticed` cannot be observed while its fixture is gone; continuation went unexercised in two of six runs; `ritual-authoring` was not exercised on Claude; `golden-files` and `title-cap` on both hosts, `commit-trailers` on Codex and `trunk-based` on Claude were exercised neither here nor at the baseline, so no comparison exists for them.
- **Disk and provider failures:** each run they cut short was repeated from an empty directory, and no record in the comparison carries either failure. The hosts' Go builds wrote to the personal `GOCACHE`, which the harness does not redirect; `du -sh ~/.cache/go-build` printed 35G when the disk filled a second time, during the interleaved arms, cutting their fifth runs short before the sessions started; both were repeated.
- **What the hosts could read:** M-0338's own body describes this observation, and M-0335's names the pilot's fixture and the reads it measured.
- **The judge** reads reduced transcripts and the run's repository; its verdicts are a reading with quoted evidence, not a test.
- **Raw records:** every run's records, Codex's session rollouts, each run's starting guidance documents and the two fixtures are kept, without the credentials Codex was given, in `.scratch/e0092-after/records.tar.xz`, gitignored, in E-0092's worktree. Unpacked, `summarize.py` reproduces the round's and each arm's tables from them byte for byte, T4's corrected Claude row aside; the L2 arms table above is assembled from its per-arm output. The clones are not kept.

### AC-2 — growth against the frozen baseline

Measured 2026-10-15 in the Linux devcontainer (go1.25.12, Python 3.12.3), in E-0092's worktree at `8fab74d78`, the last commit that changes code, with no guidance change after `942cd2957`, and with `.guidance/index.md`'s installed commit `9c9ca4b3681ca4cb8798e5af9e9124b1e761526b`. M-0334's figures are at its frozen post-delivery, pre-reduction revision `708bf3f80`.
- **One definition for both figures.** The ceiling measure's code changed after `708bf3f80` (`internal/projectguidance/` and the ceiling test). In a `git clone --no-hardlinks` of the worktree at `626f03595`, `CLAUDE.md`, `AGENTS.md`, `.guidance/project.md`, `.guidance/index.md` and `internal/skills/embedded-guidance/` were checked out from `708bf3f80` and `.guidance/project/` removed, and `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` printed Claude Code handwritten primed 9647 and aiwf-generated 2393, Codex 9776 and 2430: M-0334's figures exactly; the measure's code at `626f03595` differs from `8fab74d78`'s only in the ceiling constants. The test failed there, as expected, on the ceilings and on 17 links the old `CLAUDE.md` carries that the current read table leaves unclassified; neither reading counts those links.
- **Upfront load per host.** `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/`, expected to pass at the ceilings, printed:
  - Claude Code: handwritten primed 1292 (ceiling 1292), aiwf-generated 2460, required reads `.guidance/project.md`, `.guidance/index.md`; at the baseline 9647 and 2393.
  - Codex: handwritten primed 1432 (ceiling 1432), aiwf-generated 2497, required reads `.guidance/project.md`, `.guidance/index.md`, `CLAUDE.md`; at the baseline 9776 and 2430.
  - Both are under M-0349's targets, 1629 and 1758, which `TestE0092Inventory` logs and holds.
- **Conditional inventory.** The same command's `on demand:` lines: nine documents under `.guidance/project/`, 6,288 words, where the baseline had none; the five pack documents unchanged at 1,859 words.
- **Task-loaded words** are the round's loading table's last column. They rose in the Go tasks and most rule tasks, as M-0334 predicted once rules are read from the routed documents (Claude's T1 mean from 0 to 1,855, Codex's from 796 to 4,662); Claude's T2 and T8, and T9 on both hosts, read no routed document, as at the baseline. They also rose on L1 and L2, where it predicted no change: Claude 168 → 653 and 103 → 485, Codex 679 → 1,101 and 665 → 1,572. The L1 and L2 rise comes from documents read for work they only partly serve: Codex opened `environment.md`, whose router entry names subagent worktrees, together with the `aiwf-worktree` skill when it made its branch, and both hosts opened `testing.md`, which holds Go-specific and general development rules together, for a test that is not Go. M-0361 separates them.
- **Total load per task** fell in every task on both hosts, by 4,428 words (Codex T1, 13,002 → 8,574) to 8,305 (Claude T2, T8 and T9, 12,040 → 3,735): each host's handwritten primed and aiwf-generated words from the ceiling test at the round's revision, `87c4b20b7` (1,275 + 2,460 and 1,415 + 2,497), and at the baseline's, plus the task's mean task-loaded words from that round's loading table.
- **Policy and test growth.** `scripts/growth-report.py --at 8fab74d78 --baseline 708bf3f80` exited 0: test:production 2.36 against 2.34, policy chokepoints 96 against 97, policy lines 63,153 against 63,069 (67.7% of production against 67.4%), production lines 93,327 against 93,611, test lines 219,811 against 219,363, shipped skill and guidance words 88,468 against 88,178.
- **What rose:** M-0334 recorded no expected direction for these metrics. Against the baseline, the aiwf-generated load rose by 67 words per host and shipped words by 290, where M-0336 moved rules into the shipped fragment and skills; policy lines rose by 84 and test lines by 448, with test:production from 2.34 to 2.36, while production lines fell by 284. On L1 and L2 the task-loaded rise is a regression against M-0334's expected direction, no change, and M-0361 takes up its cause; in the other tasks it is the direction M-0334 predicted. The trailing-thirty-day instruction-commit count M-0334 recorded is not part of this comparison.

### AC-3 — every inventory decision carried out

`go test -count=1 -run 'TestE0092Inventory$|TestInventoryReconcileFindings$' -v ./internal/policies/` at `8fab74d78`, in E-0092's worktree, go1.25.12, expected to pass, printed `--- PASS: TestInventoryReconcileFindings` and `--- PASS: TestE0092Inventory` with `ok`. The reconciliation reports a row whose decision changes text when no block carries it out: one naming the row, not as an id edit, with its decided disposition, in any commit since M-0336's start, trunk included (`2d3799945`, `7b856039a`, `8fab74d78`); on this tree it reports none. In a clone at `7b856039a` with `origin/main` moved to `HEAD`, the same test passed. Run again at `0204e63df` on a clean tree, it passed.

### Gates

`make check-fast` at `8fab74d78`, in E-0092's worktree, exited 0: `golangci-lint` printed `0 issues.` and all 73 test packages printed `ok`. An earlier run at `5e0844f91`, while other test suites ran beside it, failed only `internal/check`'s `TestFSMHistoryConsistent_PerfBudget` on its 10-second wall-clock budget, the sensitivity to load G-0730 records.

## Deferrals

- Go's pack reach and the documents read for work they only partly serve: Go-specific development rules separated from general ones and the router's entries named by activity, in M-0361 (Decisions made during implementation).
- Whether the builder agent's Sonnet tier follows the reduced guidance: G-0737.

## Reviewer notes

- (none)
