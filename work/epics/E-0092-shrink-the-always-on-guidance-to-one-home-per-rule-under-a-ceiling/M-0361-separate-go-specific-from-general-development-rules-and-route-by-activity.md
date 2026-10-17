---
id: M-0361
title: Separate Go-specific from general development rules and route by activity
status: in_progress
parent: E-0092
depends_on:
    - M-0338
tdd: none
acs:
    - id: AC-1
      title: The reconciliation judges only blocks after M-0338's end
      status: open
    - id: AC-2
      title: Each on-demand development rule sits where its M-0349 row places it
      status: open
    - id: AC-3
      title: Before and after, each host reaches the documents its task needs
      status: open
---
## Goal

Separate this repository's Go-specific development rules from its general ones, so each task reads only what serves it: general development rules in documents named by the activity they serve, Go-specific ones as this repository's overrides of the Go pack, read with it, and router entries that name activities. A before-and-after observation shows whether each host now reaches the documents its task needs and only those.

## Closes

- (none)

## Context

M-0338's after-observation found two faults in how M-0335 grouped the on-demand development rules, by topic rather than by language and activity:
- **Go is the one language with a project document on the router**, an entry presenting `go-conventions.md` as overriding the pack, and `go-conventions.md` repeats part of the pack. Claude read the Go pack in no Go run of the round, where it read `go-conventions.md` before its first edit in 3 of 12; in five T3 runs with the router's pack route it read `go-conventions.md` in all five and the pack in one. Codex read both in every Go run.
- **Documents are read for work they only partly serve.** `testing.md` holds Go-specific rules (test discipline, the coverage gates, the `go test` wrappers) together with general ones (test design, an acceptance criterion's evidence), and hosts opened it for Python and TypeScript tests; Codex also read it before its first edit in every T2, T4, T7 and T8 run. Codex opened `environment.md`, whose router entry names subagent worktrees, whenever it made a worktree for its branch. On L1 and L2 the task-loaded words rose against a predicted no change, though every task's total load fell.

Making `go-conventions.md` primed was ruled out, since it takes the primed load over M-0349's targets and makes Go the one language always loaded. The audience split between shipped and repository guidance is not in question: no Go-specific rule ships. M-0338's Validation and Decisions made during implementation hold the measurements.

## Acceptance criteria

### AC-1 — The reconciliation judges only blocks after M-0338's end

The reconciliation judges the disposition blocks of commits after M-0338's final commit, `24384e301`; each earlier block was judged green while its milestone was open, so a row whose decision this milestone changes leaves the block that carried out the earlier decision unjudged. **Pass criterion**: `TestE0092Inventory` passes once this milestone has changed a row's decision and a commit has carried out the new one, where judging every block since the reconciliation base reports the earlier block; the completeness rule still requires, from every commit since that base, a commit carrying out each row's decision at HEAD. **Edge cases**: a row changed twice within this milestone reports its first block, so each row changes once; a malformed block before that commit is no longer reported, and at `b7ada45ee` none existed. **Code references**: `repoRemovalBlocks` and `TestE0092Inventory` in `internal/policies/e0092_inventory_test.go`.

### AC-2 — Each on-demand development rule sits where its M-0349 row places it

Each development rule M-0349's inventory routes on demand sits in the document its row names: a general rule in a document whose router entry names the activity it serves, a Go-specific rule only in this repository's overrides of the Go pack, which the router reaches after the pack, and no rule there restating the pack. Each changed placement or deletion is a maintainer decision recorded on its M-0349 row, with M-0349's router draft changed to match, and carried out by a commit whose disposition block names the row. Whether a placement is right is judged at that decision and by AC-3; the test holds the tree to the decisions. **Pass criterion**: at this milestone's last guidance commit `TestE0092Inventory` passes, so every destination is a document the router draft routes, every decision has a commit carrying it out and every block matches its row, and `PolicyGuidanceCeiling` holds both hosts' primed load at their ceilings, under M-0349's targets. **Edge cases**: a live reference to a moved or renamed document is re-pointed, and for a Go comment no check reports a miss; a router reword that adds primed words needs a ceiling raised by maintainer decision. **Code references**: M-0349's inventory and router draft; `.guidance/project.md` and `.guidance/project/`; `internal/policies/e0092_inventory_test.go` and `internal/policies/guidance_ceiling.go`.

### AC-3 — Before and after, each host reaches the documents its task needs

Both hosts are observed before and after this milestone's guidance changes, with `scripts/e0092-observe/` at its pin and M-0334's rubric as committed. **Pass criterion**: Validation records, for T1, T3, T7, T8, T11, L1 and L2 on Claude Code and Codex at three runs each, a baseline at M-0338's final commit and an after-run at this milestone's last guidance commit: the documents each task is expected to need, written into this spec before the after-run starts; each run's reads before its first edit; each task's loaded words; and the judge's verdicts, with no rule that held on a host at the baseline violated after. It also records the ceiling measure and the growth report at that commit as a row of `docs/design/growth.md`. Each record carries the command, the expectation, the observation and the environment: host versions and settings, the checkout and the guidance revision. **Edge cases**: a run lost to a provider error is redone and named; a first edit `shapes_scan.py` lists is judged by reading the run; an observation not made stays outstanding. **Code references**: `scripts/e0092-observe/run.sh`, `judge.py`, `summarize.py` and `shapes_scan.py`.

## Constraints

- Removing text from `go-conventions.md` changes M-0349 row decisions, so each change is a maintainer decision recorded in M-0349, and the inventory test's reconciliation must accept the blocks that carried a row's earlier decision.
- The observation reuses `scripts/e0092-observe/` at its pin and M-0334's rubric as committed, with a baseline at M-0338's final commit for T1, T3, T7, T8, T11, L1 and L2 on both hosts.
- `guidance.enabled` stays `false` until this milestone completes.
- The harness leaves the hosts' Go build cache at the personal `GOCACHE`; during M-0338, `du -sh ~/.cache/go-build` printed 35G when the disk filled a second time. Free it before a round.
- The obligations M-0334's Reviewer notes place on a run of the harness bind this milestone's runs as they bound M-0338's.
- This milestone changes no file the binary embeds, so nothing it moves ships to a consumer. The wrap records `git diff --stat` over the milestone's range for `internal/skills/embedded*`, `internal/recipe/embedded` and `internal/htmlrender/embedded` under Validation.

## Design notes

- M-0338's Decisions made during implementation hold the direction, decided by the maintainer on 2026-10-15 and 2026-10-16. Each row's new placement is a maintainer decision taken when this milestone starts and recorded on its M-0349 row.

## Out of scope

- Changing the Go pack itself; its wording belongs to the external source.
