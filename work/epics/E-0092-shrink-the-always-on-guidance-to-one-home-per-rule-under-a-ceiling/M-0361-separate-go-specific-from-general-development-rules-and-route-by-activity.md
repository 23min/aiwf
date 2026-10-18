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
      status: met
    - id: AC-2
      title: Each on-demand development rule sits where its M-0349 row places it
      status: met
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

Each development rule M-0349's inventory routes on demand sits in the document its row names: a general rule in a document whose router entry names the activity it serves, a Go-specific rule only in this repository's Go conventions document, which the router reaches after the Go pack and which repeats from the pack only the rules this repository depends on. Each changed placement or deletion is a maintainer decision recorded on its M-0349 row, with M-0349's router draft changed to match, and carried out by a commit whose disposition block names the row. Whether a placement is right is judged at that decision and by AC-3; the test holds the tree to the decisions. **Pass criterion**: at this milestone's last guidance commit `TestE0092Inventory` passes, so every destination is a document the router draft routes, every decision has a commit carrying it out and every block matches its row, and `PolicyGuidanceCeiling` holds both hosts' primed load at their ceilings, under M-0349's targets. **Edge cases**: a live reference to a moved or renamed document is re-pointed, and for a Go comment no check reports a miss; a router reword that adds primed words needs a ceiling raised by maintainer decision. **Code references**: M-0349's inventory and router draft; `.guidance/project.md` and `.guidance/project/`; `internal/policies/e0092_inventory_test.go` and `internal/policies/guidance_ceiling.go`.

### AC-3 — Before and after, each host reaches the documents its task needs

Both hosts are observed before and after this milestone's guidance changes, with `scripts/e0092-observe/` at its pin and M-0334's rubric as committed. **Pass criterion**: Validation records, for T1, T3, T7, T8, T11, L1 and L2 on Claude Code and Codex at three runs each, a baseline at M-0338's final commit and an after-run at the router's commit, and T1 on Claude repeated at each later commit that changes the guidance: the documents each task is expected to need, written into this spec before the after-run starts; each run's reads before its first edit; each task's loaded words; and the judge's verdicts, with every violation of a rule that held on a host at the baseline recorded with its vote, what the run had read, and the maintainer's judgement of it. It also records the ceiling measure and the growth report at the last commit that changes the guidance as a row of `docs/design/growth.md`. Each record carries the command, the expectation, the observation and the environment: host versions and settings, the checkout and the guidance revision. **Edge cases**: a run lost to a provider error, a host that changes version mid-round, or a timeout the maintainer chooses to redo is redone and named, the lost run kept; a first edit `shapes_scan.py` lists is judged by reading the run; an observation not made stays outstanding. **Code references**: `scripts/e0092-observe/run.sh`, `judge.py`, `summarize.py` and `shapes_scan.py`.

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

## Decisions made during implementation

Each was decided by the maintainer, on 2026-10-17 or, for the table rule and the restated clauses, on 2026-10-18.

- **The reconciliation judges only the blocks after M-0338's end.** Judging each block against its row as the block's own commit left it was rejected: it reads M-0349's history to re-judge blocks already judged green while their milestones were open.
- **One Go document.** The Go-specific testing rules join `go-conventions.md` rather than a Go testing document of their own: most Go tasks write tests and would read both, and a second router entry adds primed words.
- **The reconciliation accepts a pack-copy block for a moved rule.** Most of the Go overlaps are clauses inside rules that also say something the pack does not, so the reconciliation accepts a `copy of` a pack block for a row moved on demand, and that block carries nothing out.
- **Golden files and testing every branch stay general.** Each holds in any language once the Go pack's `testdata/` clause leaves; M-0349's rows record where every other testing rule went.
- **The table rule is stated for tests in any language in `testing.md`.** The after-run showed Claude meeting `testing.md` in Go tasks more often than the Go pack, and the rule's multi-case half holds in any language.
- **The clauses trimmed as copies of the Go pack are stated again in `go-conventions.md`.** Claude reads `go-conventions.md` in nearly every Go task and the pack in about half, so a rule only the pack states can go unread; the rest of the pack stays read on demand.
- **The router reaches the Go overrides from the language-pack line.** A Go entry of its own would keep Go the one language with a router line, and an entry presenting `go-conventions.md` as overriding the pack is the shape Claude read instead of the pack in M-0338.

## Validation

### AC-3 — the documents each task needs

Written before the after-run, which runs at `74e8fbd2f`, the last commit to change the guidance. A run needs these documents before its first edit:

| Task | Needs | Needs none of |
|---|---|---|
| T1 | the Go pack, `go-conventions.md`, `testing.md`, `verb-design.md` (changing a verb), `environment.md` (checking the flag with a built binary) | — |
| T3 | the Go pack, `go-conventions.md`, `testing.md` | `environment.md`, `verb-design.md` |
| T7 | `environment.md` | `testing.md`, the Go pack, `go-conventions.md` |
| T8 | `testing.md` (a criterion's evidence) | the Go pack, `go-conventions.md`, `environment.md` |
| T11 | the Go pack, `go-conventions.md`, `testing.md`, `verb-design.md` (changing a verb's output) | `environment.md` |
| L1 | the Python pack, `testing.md` | the Go pack, `go-conventions.md`, `environment.md` |
| L2 | the TypeScript pack, `testing.md` | the Go pack, `go-conventions.md`, `environment.md` |

At `74e8fbd2f`, five judged rules had part of their text only in the Go pack: `t-parallel` and `table-driven` (T1), the first-parameter half of `context-first` and the code range of `exit-codes` (T1), and the `testdata/` half of `golden-files` (T11). A run that skips the pack can miss them; each is judged as before.

### AC-3 — the observation

- **Environment.** The Linux devcontainer, two runs at a time for the rounds and one at a time for the T1 repeats. Every Claude run reports Claude Code 2.1.296 in its init event and every Codex run counted below 0.163.0 as its rollout's `cli_version`; the judge is `claude-opus-5-5` at effort `xhigh`, three calls per run, the majority standing. `git log 5092de5fd.. -- scripts/e0092-observe/` is empty at every checkout run, and M-0334's Loading observations and Rule tasks hash to `2b5b146d…` at each.
- **Commands.** Each run is `OBSERVE_DIR=<scratch>/<round> scripts/e0092-observe/run.sh <checkout> <host> <task> <n>`, then `scripts/e0092-observe/judge.py <run>`; the tables are `summarize.py <round>`, and `shapes_scan.py <round>` lists only commands that write nothing.
- **Rounds.** The baseline at `24384e301` from 2026-10-17T10:56Z and the after-run at `74e8fbd2f` from 14:19Z, each the seven tasks on both hosts at three runs; T1 runs 4–6 on both hosts at both commits from 18:39Z; T1 on Claude six times at `e92ce85e0` and six times at `e8d1d495c` on 2026-10-18.
- **Runs redone.** Codex installed 0.163.0 over 0.162.1 at 2026-10-17T11:54:39Z, during the baseline, so its four Codex runs that started on 0.162.1 (T1, T3, T7 and T8, run 1) were redone on 0.163.0 and the originals kept aside. The after-run's T1/claude-1 reached the 1,800-second timeout during its own mutation checks and was redone, the timed-out run kept. The records of every round, the set-aside runs included, are in `.scratch/e0092-m0361/records.tar.xz`.
- **Reads before the first edit, against the expectation above.** Claude read the Go pack in no Go run at the baseline (0 of 12) and in 9 of 24 after the change; it read `go-conventions.md` in 3 of 12 and 20 of 24, and opened the router in 10 of 12 and 22 of 24. Claude read the TypeScript pack in 1 of 3 L2 runs at the baseline and 3 of 3 after. Codex read `environment.md` in every L1 and L2 run at the baseline and in 1 and 2 of 3 after, and in T7 read `environment.md` alone, where at the baseline it also read `testing.md`. No host read a Go document in L1, L2, T7 or T8. Both hosts read `environment.md` in every T3 run, which the expectation did not predict.
- **Rules that held at the baseline and were violated after.** At `74e8fbd2f`:
  - `table-driven` on Claude: violated in 2 of the 3 T1 runs among runs 1–6 that wrote code (votes 2/3 and 3/3), against none of 6 at the baseline. One violating run had read the Go pack, the other `testing.md` alone, which no longer stated the rule. With the rule stated for any language in `testing.md`, none of the 6 runs at `e92ce85e0` and none of the 5 at `e8d1d495c` that wrote code violated it.
  - `table-driven` on Codex: 1 of 6 at the baseline (2/3) and 1 of 6 after (3/3).
  - `naming` on Codex: T1/codex-1, 2/3, against none of 6 at the baseline; the run had read `go-conventions.md`, which states the rule unchanged.
  - `tests-in-devcontainer` on Codex: T3/codex-3, 3/3, against none of 3 at the baseline; the run had read `go-conventions.md`, which holds the rule moved word for word.
  
  M-0334's baseline and M-0338's round both record Codex violating `naming` and `tests-in-devcontainer`. The maintainer judged on 2026-10-18 that those two are Codex's run-to-run variation rather than an effect of this milestone, and that the `table-driven` loss is corrected by stating the rule in `testing.md`. Every other rule that held at the baseline held after.
- **Not held at the baseline.** Codex `context-first` was violated in 2 of 6 T1 runs before and after. Claude `contract-tests`, `operator-setup` and `spec-sourced-inputs` and Codex `settings-consent` and `spec-sourced-inputs` were violated at the baseline and after.
- **Runs that stopped to ask.** Claude stopped T1 to ask a design question before writing code in none of 6 baseline runs, 3 of 6 at `74e8fbd2f`, none of 6 at `e92ce85e0` and 1 of 6 at `e8d1d495c`; those runs exercise no T1 rule. The after-change runs read `verb-design.md`, which the router now names for changing a verb, in most runs, where the baseline read it in 1 of 6; whether that causes the stops is not measured.
- **Loaded words.** Task-loaded words fell for Codex T7 (1,876 → 422) and T8 (3,168 → 2,256) and rose where a task now reads `testing.md` or the Go documents, Claude L1 among them (168 → 710).
- **Growth.** `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` at `19982ca38`, whose guidance and policy code match `e8d1d495c`, printed handwritten primed 1288 (ceiling 1288) for Claude Code and 1428 (ceiling 1428) for Codex; `scripts/growth-report.py --at e8d1d495c --baseline 24384e301` printed policy lines 63,153 → 63,167 and test lines 219,811 → 219,825, everything else unchanged. Recorded as the 2026-10-18 row of `docs/design/growth.md`.
