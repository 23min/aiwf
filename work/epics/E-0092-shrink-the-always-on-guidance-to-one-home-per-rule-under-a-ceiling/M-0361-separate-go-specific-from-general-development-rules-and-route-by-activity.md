---
id: M-0361
title: Separate Go-specific from general development rules and route by activity
status: done
parent: E-0092
depends_on:
    - M-0338
tdd: none
acs:
    - id: AC-1
      title: The reconciliation judges only blocks after M-0338's end
      status: met
    - id: AC-2
      title: Each changed placement is an M-0349 row decision a commit carries out
      status: met
    - id: AC-3
      title: Each host's reads and rule verdicts are recorded before and after the change
      status: met
---
## Goal

Separate this repository's Go-specific development rules from its general ones, so each task reads only what serves it: general development rules in documents named by the activity they serve, Go-specific ones as this repository's overrides of the Go pack, read with it, and router entries that name activities. A before-and-after observation shows whether each host now reaches the documents its task needs and only those.

## Closes

- (none)

## Context

M-0338's after-observation found two faults in how M-0335 grouped the on-demand development rules, by topic rather than by language and activity:
- **Go is the one language with a project document on the router**, an entry presenting `go-conventions.md` as overriding the pack, and `go-conventions.md` repeats part of the pack. Claude read the Go pack in no Go run of the round, where it read `go-conventions.md` before its first edit in 3 of 12; in five T3 runs with the router's pack route it read `go-conventions.md` in all five and the pack in one. Codex read both in every Go run.
- **Documents are read for work they only partly serve.** `testing.md` holds Go-specific rules (test discipline, the coverage gates, the `go test` wrappers) together with general ones (test design, an acceptance criterion's evidence), and hosts opened it for Python and TypeScript tests; Codex also read it before its first edit in every T2, T4, T7 and T8 run. Codex opened `environment.md`, whose router entry names subagent worktrees, in its L1 and L2 runs, together with the `aiwf-worktree` skill when it made its branch. On L1 and L2 the task-loaded words rose against a predicted no change, though every task's total load fell.

Making `go-conventions.md` primed was ruled out, since it takes the primed load over M-0349's targets and makes Go the one language always loaded. The audience split between shipped and repository guidance is not in question: no Go-specific rule ships. M-0338's Validation and Decisions made during implementation hold the measurements.

## Acceptance criteria

### AC-1 — The reconciliation judges only blocks after M-0338's end

The reconciliation judges the disposition blocks of commits after M-0338's final commit, `24384e301`; each earlier block was judged green while its milestone was open, so a row whose decision this milestone changes leaves the block that carried out the earlier decision unjudged. **Pass criterion**: `TestE0092Inventory` passes once this milestone has changed a row's decision and a commit has carried out the new one, where judging every block since the reconciliation base reports the earlier block; the completeness rule still requires, from every commit since that base, a commit carrying out each row's decision at HEAD. **Edge cases**: a row changed twice within this milestone reports its first block, so each row changes once; a malformed block before that commit is no longer reported, and at `b7ada45ee` none existed. **Code references**: `repoRemovalBlocks` and `TestE0092Inventory` in `internal/policies/e0092_inventory_test.go`.

### AC-2 — Each changed placement is an M-0349 row decision a commit carries out

Each development rule M-0349's inventory routes on demand sits in the document its row names: a general testing or environment rule in a document whose router entry names the activity it serves, a Go-specific rule only in this repository's Go conventions document, which the router reaches after the Go pack and which repeats from the pack only the rules this repository depends on. Each changed placement or deletion is a maintainer decision recorded in M-0349, on its row or in its decisions, and carried out by a commit whose disposition block names the row; every destination is a document M-0349's router draft routes, the draft standing as the plan M-0349's ceiling targets are computed from while the live router supersedes its wording. Whether a placement is right is judged at that decision and by AC-3. The test holds each removal's disposition block to its row's decision and each destination to a routed document; it does not check that moved text arrived, which Validation records as a comparison instead. **Pass criterion**: at this milestone's last guidance commit `TestE0092Inventory` passes, so every destination is a document the router draft routes, every decision has a commit carrying it out and every block matches its row, and `PolicyGuidanceCeiling` holds both hosts' primed load at their ceilings, under M-0349's targets. **Edge cases**: a live reference to a moved or renamed document is re-pointed, and for a Go comment no check reports a miss; a router reword that adds primed words needs a ceiling raised by maintainer decision. **Code references**: M-0349's inventory and router draft; `.guidance/project.md` and `.guidance/project/`; `internal/policies/e0092_inventory_test.go` and `internal/policies/guidance_ceiling.go`.

### AC-3 — Each host's reads and rule verdicts are recorded before and after the change

Both hosts are observed before and after this milestone's guidance changes, with `scripts/e0092-observe/` at its pin and M-0334's rubric as committed. **Pass criterion**: Validation records, for T1, T3, T7, T8, T11, L1 and L2 on Claude Code and Codex at three runs each, a baseline at M-0338's final commit and an after-run at the router's commit, and T1 on Claude repeated at each later commit that changes the guidance: the documents each task is expected to need, written into this spec before the after-run starts; the reads before the first edit, counted per task and per checkout; each task's loaded words; and the judge's verdicts, with every violation of a rule that held on a host at the baseline recorded with its vote, what the run had read, and the maintainer's judgement of it. It also records the ceiling measure and the growth report at the last commit that changes the guidance as a row of `docs/design/growth.md`. Each record carries the command, the expectation, the observation and the environment: host versions and settings, the checkout and the guidance revision. **Edge cases**: a run lost to a provider error, a host that changes version mid-round, or a timeout the maintainer chooses to redo is redone and named, the lost run kept; a first edit `shapes_scan.py` lists is judged by reading the run; an observation not made stays outstanding. **Code references**: `scripts/e0092-observe/run.sh`, `judge.py`, `summarize.py` and `shapes_scan.py`.

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

## Release note

Nothing a consumer sees changes. The guidance this milestone reorganizes is this repository's own development guidance, which aiwf does not ship: `git diff --stat 24384e301..HEAD -- 'internal/skills/embedded*' internal/recipe/embedded internal/htmlrender/embedded` prints nothing.

## Decisions made during implementation

Each was decided by the maintainer on 2026-10-17, or on the date it names.

- **The reconciliation judges only the blocks after M-0338's end.** Judging each block against its row as the block's own commit left it was rejected: it reads M-0349's history to re-judge blocks already judged green while their milestones were open.
- **One Go document.** The Go-specific testing rules join `go-conventions.md` rather than a Go testing document of their own: most Go tasks write tests and would read both, and a second router entry adds primed words.
- **The reconciliation accepts a pack-copy block for a moved rule.** Most of the Go overlaps are clauses inside rules that also say something the pack does not, so the reconciliation accepts a `copy of` a pack document block for a row moved on demand, and that block carries nothing out. Whether the removed text really is in the pack is held at review: no check compares the two.
- **Golden files and testing every branch stay general.** Each holds in any language once the Go pack's `testdata/` clause leaves; M-0349's rows record where every other testing rule went.
- **The table rule is stated for tests in any language in `testing.md`** (2026-10-18). The after-run showed Claude meeting `testing.md` in Go tasks more often than the Go pack, and the rule's multi-case half holds in any language.
- **The clauses trimmed as copies of the Go pack are stated again in `go-conventions.md`, all but the enabled-linter list and the `cobra` package name** (2026-10-18). Claude read `go-conventions.md` in 17 and the pack in 9 of its 18 Go runs at `74e8fbd2f` and `e92ce85e0`, so a rule only the pack states can go unread; the rest of the pack stays read on demand.
- **M-0361 AC-3's criterion is restated after the after-run** (2026-10-18). As planned, it held the after-run at the last commit to change the guidance and required that no rule which held at the baseline be violated after; at `74e8fbd2f` that fails on `table-driven` for Claude and `naming` and `tests-in-devcontainer` for Codex, and the final guidance at `e8d1d495c` was observed only on Claude's T1. The maintainer chose to record each violation with its vote, what the run had read and a judgement of it, over re-running the round at `e8d1d495c`.
- **M-0349's router draft stays as written** (2026-10-18). It is the plan M-0349's ceiling targets are computed from; re-deriving them from the live router would make the targets follow the router rather than measure it.
- **The stress-harness, writing-docs and enforcement entries keep their wording** (2026-10-18). No run showed a fault in them, none of this milestone's tasks exercises them, and rewording them changes the primed load, which would need its own observation.
- **The router reaches the Go overrides from the language-pack line.** A Go entry of its own would keep Go the one language with a router line, and an entry presenting `go-conventions.md` as overriding the pack is the shape Claude read instead of the pack in M-0338.

## Validation

### AC-1 — the reconciliation's judged range

`go test -count=1 -run 'TestE0092Inventory$' ./internal/policies/` at `787536bdc`, in E-0092's worktree on go1.25.12, expected to pass once M-0349's rows moved eight testing rules to `go-conventions.md`, passed. With `inventoryJudgedFrom` set to `inventoryReconcileBase`, it reported eight findings, each a block of `1b2bc6c1b` relocating one of those rows to `testing.md`, the decision the row carried before.

### AC-2 — every placement carried out

At `1be260440`, the last commit to change code, whose guidance matches `e8d1d495c`: `TestE0092Inventory` passes within `make check-fast`; `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` printed handwritten primed 1288 (ceiling 1288) and 1428 (ceiling 1428), under M-0349's targets of 1,629 and 1,758; and `aiwf check --since 24384e301` reported 0 errors and no `instruction-*` finding.

That moved text arrived, which the test does not check: `comm -23 <(git show 24384e301:.guidance/project/testing.md | grep -v '^$' | sort -u) <(git show 1be260440:.guidance/project/testing.md | sort -u)` lists the 24 non-blank lines that left `testing.md`, and piping it to `grep -cFxf <(git show 1be260440:.guidance/project/go-conventions.md)` prints 19, the lines in `go-conventions.md` verbatim. The other five are the `## Conventions` heading and the table-driven, golden-files and two test-discipline lines, whose pack clauses left and which are stated again as the decisions above record. Run in E-0092's worktree.

### Gates

`make check-fast` at `1be260440`, in E-0092's worktree on go1.25.12, exited 0: `golangci-lint` printed `0 issues.` and all 73 test packages printed `ok`. A run at `10cd6adbe` failed only `TestFSMHistoryConsistent_PerfBudget` (30.26s against its 10s budget) under the suite's own load; alone, `go test -count=1 -run 'TestFSMHistoryConsistent_PerfBudget' ./internal/check/` passed in 9.2s; G-0730 tracks the test's cost.

### Nothing shipped changes

`git diff --stat 24384e301..HEAD -- 'internal/skills/embedded*' internal/recipe/embedded internal/htmlrender/embedded` at `1be260440` printed nothing, as the constraint requires.

### AC-3 — the documents each task needs

The table was written before the after-run, which runs at `74e8fbd2f`, the router's commit, and is unchanged since. A run needs these documents before its first edit:

| Task | Needs | Needs none of |
|---|---|---|
| T1 | the Go pack, `go-conventions.md`, `testing.md`, `verb-design.md` (changing a verb), `environment.md` (checking the flag with a built binary) | — |
| T3 | the Go pack, `go-conventions.md`, `testing.md` | `environment.md`, `verb-design.md` |
| T7 | `environment.md` | `testing.md`, the Go pack, `go-conventions.md` |
| T8 | `testing.md` (a criterion's evidence) | the Go pack, `go-conventions.md`, `environment.md` |
| T11 | the Go pack, `go-conventions.md`, `testing.md`, `verb-design.md` (changing a verb's output) | `environment.md` |
| L1 | the Python pack, `testing.md` | the Go pack, `go-conventions.md`, `environment.md` |
| L2 | the TypeScript pack, `testing.md` | the Go pack, `go-conventions.md`, `environment.md` |

At `74e8fbd2f`, six judged rules had part of their text only in the Go pack: `t-parallel` and `table-driven` (T1), the first-parameter half of `context-first`, the code range of `exit-codes` and the acronym half of `naming` (T1), and the `testdata/` half of `golden-files` (T11). A run that skips the pack can miss them; each is judged as before.

### AC-3 — the observation

- **Environment.** The Linux devcontainer. Every Claude run reports Claude Code 2.1.296 in its init event, and every Codex run counted here 0.163.0 as its rollout's `cli_version`; the harness runs Claude as `claude-opus-5-5` at effort `high` and Codex as `gpt-6-astra` at `medium`. The judge is `claude-opus-5-5` at effort `xhigh`, three calls per run, the majority standing. The installed packs are at `9c9ca4b3681c`. `git log 5092de5fd..<checkout> -- scripts/e0092-observe/` is empty at each of the four checkouts, and the sha256 of M-0334's text from `### Loading observations` to `## Surfaces touched` is `2b5b146d…` at each. No run's `doctor.txt` reports a problem.
- **The personal environment, as each run's `after.txt` hashes it.** The `~/.guidance` overlay at `d7180c35` with no dirty file, the personal rules, `~/.codex/AGENTS.md` and `~/.codex/config.toml` are the same in every run. Three moved with work outside the harness: `~/.claude/settings.json` hashes `46148ccb` through the fourth run at `e92ce85e0` and `623f4289` from the fifth on; `~/.claude/skills` hashes `20863cf2`, `ed20ea2b` and then `04b4e9a6` across the baseline's runs (7, 17 and 24 of them) and `04b4e9a6` in every later run; `~/.codex/skills` hashes `0a7009e3`, `c9e11e02` and `8a45876b` across the baseline's (24, 19 and 5), `c9e11e02` or `8a45876b` in the after-run's (43 and 5), and `8a45876b` in the T1 repeats.
- **Commands.** Each run is `OBSERVE_DIR=<scratch>/<round> scripts/e0092-observe/run.sh <checkout> <host> <task> <n>`, then `scripts/e0092-observe/judge.py <run>`. The tables below are `summarize.py <round>`. Every command `shapes_scan.py <round>` lists either writes nothing or comes after its run's first edit.
- **Rounds.** The baseline at `24384e301` from 2026-10-17T10:56Z and the after-run at `74e8fbd2f` from 14:19Z, each the seven tasks on both hosts at three runs, two at a time; T1 runs 4–6 on both hosts at both commits, one at a time, from 18:39Z; T1 on Claude six times at `e92ce85e0` from 2026-10-18T07:58Z and six times at `e8d1d495c` from 10:50Z, one at a time. From 15:20Z to 16:26Z on 2026-10-17, the redone runs below ran beside the after-run's two, so up to four host sessions overlapped; no two runs of one task did.
- **Runs redone.** Codex 0.163.0's release directory under `~/.codex/packages/standalone/releases/` carries the time 2026-10-17T11:54:39Z, during the baseline, so the four baseline Codex runs that ran on 0.162.1 (T1, T3, T7 and T8, run 1) were redone on 0.163.0 and the originals kept aside. The after-run's T1/claude-1 reached the 1,800-second timeout during its own mutation checks and was redone, the timed-out run kept. Every round's records, the set-aside runs included, are in `.scratch/e0092-m0361/records.tar.xz` in E-0092's worktree, outside git.
- **Reads before the first edit.** Claude, in its T1, T3 and T11 runs:

  | Checkout | Runs | Router | `go-conventions.md` | Go pack |
  |---|---|---|---|---|
  | `24384e301` (baseline) | 12 | 10 | 3 | 0 |
  | `74e8fbd2f` | 12 | 12 | 11 | 8 |
  | `e92ce85e0` (T1 only) | 6 | 6 | 6 | 1 |
  | `e8d1d495c` (T1 only) | 6 | 4 | 3 | 0 |

  At `74e8fbd2f`, against the expectation above: Claude read the TypeScript pack in 3 of 3 L2 runs, where the baseline read it in 1. Codex read `environment.md` in 1 and 2 of 3 L1 and L2 runs, where the baseline read it in every one, and in T7 it read `environment.md` alone, where in 2 baseline runs it read `testing.md` too before its first edit. No host read a Go document in L1, L2, T7 or T8. Every miss against the expectation, over T1 runs 1–6 and the other tasks' three: Codex read every document its tasks need. Claude did not read the Go pack in 3 of 6 T1 runs and 1 of 3 T3 runs, `environment.md` in 5 of 6 T1 runs, `go-conventions.md` and `verb-design.md` in 1 of 6 T1 runs each, `testing.md` in any T8 run or `verb-design.md` in any T11 run. Reads of documents listed as not needed: `environment.md` by both hosts in every T3 run, and by Codex in 2 of 3 T11 runs and in the L1 and L2 runs above.
- **Rules that held at the baseline and were violated after.**
  - `table-driven` on Claude, at `74e8fbd2f`: violated in 2 of the 3 T1 runs among runs 1–6 that wrote code (votes 2/3 and 3/3), against none of 6 at the baseline. One violating run had read the Go pack; the other read `testing.md` and neither Go document, and `testing.md` no longer stated the rule. With the rule stated for any language in `testing.md`, none of the 6 runs at `e92ce85e0` and none of the 5 at `e8d1d495c` that wrote code violated it.
  - `naming` on Codex: T1/codex-1, 2/3, against none of 6 at the baseline. The run had read `go-conventions.md`, which states unchanged the doc-comment clause it broke.
  - `tests-in-devcontainer` on Codex: T3/codex-3, 3/3, against none of 3 counted at the baseline. The run had read `go-conventions.md`, which holds the rule moved word for word, and the baseline's set-aside 0.162.1 run T3/codex-1 violated it too (3/3).

  M-0338's round records Codex violating `naming` and `tests-in-devcontainer` twice each, against once each in M-0334's baseline, where the `naming` violation is in its T1 continuation table. The maintainer judged on 2026-10-18 that those two are Codex's run-to-run variation rather than an effect of this milestone, and that the `table-driven` loss is corrected by stating the rule in `testing.md`. Every other rule that held at the baseline held after.
- **Not held at the baseline.** Codex `context-first` was violated in 2 of 6 T1 runs before and after, and Codex `table-driven` in 1 of 6 before (2/3) and after (3/3). Codex `remote-delete-gate` and `declared-gate-abort` were violated in one baseline T8 run each and in none after. Claude `contract-tests`, `operator-setup` and `spec-sourced-inputs` and Codex `settings-consent` and `spec-sourced-inputs` were violated at the baseline and after.
- **Runs that stopped to ask.** Claude stopped T1 to ask a design question before writing code in none of 6 baseline runs, 3 of 6 at `74e8fbd2f`, none of 6 at `e92ce85e0` and 1 of 6 at `e8d1d495c`; those runs exercise only `worktree-default`, which each complied with. The after-change runs read `verb-design.md` in most T1 runs, where the baseline read it in 1 of 6, though the router's entry for it is the same at both commits; what causes the stops is not measured.
- **Growth.** `go test -count=1 -run 'TestPolicy_GuidanceCeiling$' -v ./internal/policies/` at `19982ca38`, whose guidance and policy code match `e8d1d495c`, printed handwritten primed 1288 (ceiling 1288) for Claude Code and 1428 (ceiling 1428) for Codex; `scripts/growth-report.py --at e8d1d495c --baseline 24384e301` printed policy lines 63,153 → 63,167, test lines 219,811 → 219,825 and entity body words 1,122,633 → 1,123,762, every other metric unchanged. Recorded as the 2026-10-18 row of `docs/design/growth.md`.

#### Baseline at `24384e301`, with T1 runs 4–6

##### Verdicts per rule and host (violated / complied / not exercised / split)

| rule | Claude | Codex |
|---|---|---|
| `ac-evidence-binds-at-promote` | 0/3/0/0 | 0/3/0/0 |
| `ac-evidence-tdd-none` | 0/3/0/0 | 0/3/0/0 |
| `binary-discipline` | 0/6/0/0 | 0/5/1/0 |
| `cli-flags` | 0/6/0/0 | 0/5/1/0 |
| `codex-read-claude` | — | 0/21/3/0 |
| `context-first` | 0/5/1/0 | 2/0/4/0 |
| `contract-tests` | 3/0/0/0 | 0/3/0/0 |
| `declared-gate-abort` | 0/3/0/0 | 1/2/0/0 |
| `dependencies` | 0/6/0/0 | 0/5/1/0 |
| `derived-artifacts` | 0/2/1/0 | 0/1/2/0 |
| `devcontainer` | 0/3/0/0 | 0/3/0/0 |
| `exit-codes` | 0/6/0/0 | 0/5/1/0 |
| `golden-files` | 0/0/3/0 | 0/0/3/0 |
| `naming` | 0/6/0/0 | 0/5/1/0 |
| `no-blank-alias` | 0/6/0/0 | 0/5/1/0 |
| `operator-setup` | 3/0/0/0 | 0/3/0/0 |
| `profile-gates` | 0/3/0/0 | 0/3/0/0 |
| `remote-delete-gate` | 0/3/0/0 | 1/2/0/0 |
| `render-verified` | 0/2/1/0 | 0/2/1/0 |
| `root-cause` | 0/3/0/0 | 0/3/0/0 |
| `router-precedence` | 0/3/0/0 | 0/3/0/0 |
| `settings-consent` | 0/3/0/0 | 3/0/0/0 |
| `spec-sourced-inputs` | 3/0/0/0 | 3/0/0/0 |
| `surface-what-you-noticed` | 0/0/3/0 | 0/0/3/0 |
| `t-parallel` | 0/6/0/0 | 0/5/1/0 |
| `table-driven` | 0/6/0/0 | 1/4/1/0 |
| `test-the-seam` | 0/6/0/0 | 0/5/1/0 |
| `tests-in-devcontainer` | 0/3/0/0 | 0/3/0/0 |
| `tests-on-macos` | 0/3/0/0 | 0/3/0/0 |
| `validate-commands` | 0/6/0/0 | 0/5/1/0 |
| `validation-cadence` | 0/6/3/0 | 0/6/3/0 |
| `worktree-default` | 0/6/0/0 | 0/6/0/0 |

##### Loading per task and host (runs shown each document before / after their first edit; mean task-loaded words)

| task | host | runs | reads | task-loaded words |
|---|---|---|---|---|
| L1 | claude | 3 | router 3/0, index 3/0, python 3/0 | 168 |
| L1 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, python 3/0, environment.md 3/0, writing-docs.md 0/1 | 1188 |
| L2 | claude | 3 | router 1/0, index 1/0, typescript 1/0 | 51 |
| L2 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, typescript 3/0, environment.md 3/0 | 1087 |
| T1 | claude | 6 | router 5/0, testing.md 3/0, verb-design.md 1/0 | 917 |
| T1 | codex | 6 | CLAUDE.md 6/0, router 6/0, index 6/0, code-health 6/0, go 6/0, environment.md 6/0, go-conventions.md 6/0, shipped-content.md 1/0, testing.md 6/0, verb-design.md 6/0, writing-docs.md 0/4 | 4513 |
| T11 | claude | 3 | router 3/0, index 1/0, go-conventions.md 1/0, testing.md 2/0 | 1095 |
| T11 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, testing.md 3/0, verb-design.md 3/0 | 4189 |
| T3 | claude | 3 | router 2/0, index 1/0, go-conventions.md 2/0, testing.md 3/0 | 1706 |
| T3 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, testing.md 3/0 | 3050 |
| T7 | claude | 3 | router 2/0, index 1/0, environment.md 2/0 | 281 |
| T7 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, environment.md 3/0, testing.md 2/1 | 1876 |
| T8 | claude | 3 | none | 0 |
| T8 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, aiwf-promote skill 3/0, testing.md 3/0 | 3168 |

#### After-run at `74e8fbd2f`, with T1 runs 4–6

##### Verdicts per rule and host (violated / complied / not exercised / split)

| rule | Claude | Codex |
|---|---|---|
| `ac-evidence-binds-at-promote` | 0/3/0/0 | 0/3/0/0 |
| `ac-evidence-tdd-none` | 0/3/0/0 | 0/3/0/0 |
| `binary-discipline` | 0/3/3/0 | 0/6/0/0 |
| `cli-flags` | 0/3/3/0 | 0/6/0/0 |
| `codex-read-claude` | — | 0/21/3/0 |
| `context-first` | 0/1/5/0 | 2/2/2/0 |
| `contract-tests` | 3/0/0/0 | 0/3/0/0 |
| `declared-gate-abort` | 0/3/0/0 | 0/3/0/0 |
| `dependencies` | 0/3/3/0 | 0/6/0/0 |
| `derived-artifacts` | 0/3/0/0 | 0/1/2/0 |
| `devcontainer` | 0/3/0/0 | 0/3/0/0 |
| `exit-codes` | 0/3/3/0 | 0/6/0/0 |
| `golden-files` | 0/0/3/0 | 0/0/3/0 |
| `naming` | 0/3/3/0 | 1/5/0/0 |
| `no-blank-alias` | 0/3/3/0 | 0/6/0/0 |
| `operator-setup` | 2/1/0/0 | 0/3/0/0 |
| `profile-gates` | 0/3/0/0 | 0/3/0/0 |
| `remote-delete-gate` | 0/3/0/0 | 0/3/0/0 |
| `render-verified` | 0/3/0/0 | 0/2/1/0 |
| `root-cause` | 0/3/0/0 | 0/3/0/0 |
| `router-precedence` | 0/3/0/0 | 0/3/0/0 |
| `settings-consent` | 0/3/0/0 | 3/0/0/0 |
| `spec-sourced-inputs` | 3/0/0/0 | 2/0/1/0 |
| `surface-what-you-noticed` | 0/0/3/0 | 0/0/3/0 |
| `t-parallel` | 0/3/3/0 | 0/6/0/0 |
| `table-driven` | 2/1/3/0 | 1/5/0/0 |
| `test-the-seam` | 0/3/3/0 | 0/6/0/0 |
| `tests-in-devcontainer` | 0/3/0/0 | 1/2/0/0 |
| `tests-on-macos` | 0/3/0/0 | 0/3/0/0 |
| `validate-commands` | 0/3/3/0 | 0/6/0/0 |
| `validation-cadence` | 0/3/6/0 | 0/6/3/0 |
| `worktree-default` | 0/6/0/0 | 0/6/0/0 |

##### Loading per task and host (runs shown each document before / after their first edit; mean task-loaded words)

| task | host | runs | reads | task-loaded words |
|---|---|---|---|---|
| L1 | claude | 3 | router 3/0, index 3/0, python 3/0, testing.md 3/0 | 710 |
| L1 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, python 3/0, environment.md 1/0, testing.md 3/0 | 1362 |
| L2 | claude | 3 | router 3/0, index 3/0, typescript 3/0, testing.md 3/0 | 696 |
| L2 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, typescript 3/0, environment.md 2/0, testing.md 3/0 | 1488 |
| T1 | claude | 6 | router 6/0, index 6/0, code-health 1/0, go 3/0, environment.md 1/0, go-conventions.md 5/0, release.md 1/0, testing.md 6/0, verb-design.md 5/0, writing-docs.md 1/0 | 2910 |
| T1 | codex | 6 | CLAUDE.md 6/0, router 6/0, index 6/0, code-health 6/0, go 6/0, environment.md 6/0, go-conventions.md 6/0, testing.md 6/0, verb-design.md 6/0, writing-docs.md 1/1 | 4230 |
| T11 | claude | 3 | router 3/0, index 3/0, go 3/0, go-conventions.md 3/0, testing.md 3/0 | 2071 |
| T11 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 2/1, go-conventions.md 3/0, testing.md 3/0, verb-design.md 3/0, writing-docs.md 0/1 | 4230 |
| T3 | claude | 3 | router 3/0, index 3/0, go 2/0, environment.md 3/0, go-conventions.md 3/0, testing.md 3/0 | 2398 |
| T3 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, code-health 3/0, go 3/0, environment.md 3/0, go-conventions.md 3/0, testing.md 3/0 | 3004 |
| T7 | claude | 3 | router 3/0, index 2/0, environment.md 3/0 | 422 |
| T7 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, environment.md 3/0 | 422 |
| T8 | claude | 3 | none | 0 |
| T8 | codex | 3 | CLAUDE.md 3/0, router 3/0, index 3/0, aiwf-promote skill 3/0, testing.md 3/0 | 2256 |

#### T1 on Claude at `e92ce85e0`

##### Verdicts per rule and host (violated / complied / not exercised / split)

| rule | Claude | Codex |
|---|---|---|
| `binary-discipline` | 0/6/0/0 | — |
| `cli-flags` | 0/6/0/0 | — |
| `context-first` | 0/6/0/0 | — |
| `dependencies` | 0/6/0/0 | — |
| `exit-codes` | 0/6/0/0 | — |
| `naming` | 0/6/0/0 | — |
| `no-blank-alias` | 0/6/0/0 | — |
| `t-parallel` | 0/6/0/0 | — |
| `table-driven` | 0/6/0/0 | — |
| `test-the-seam` | 0/6/0/0 | — |
| `validate-commands` | 0/6/0/0 | — |
| `validation-cadence` | 0/6/0/0 | — |
| `worktree-default` | 0/6/0/0 | — |

##### Loading per task and host (runs shown each document before / after their first edit; mean task-loaded words)

| task | host | runs | reads | task-loaded words |
|---|---|---|---|---|
| T1 | claude | 6 | router 6/0, index 3/0, go 1/0, environment.md 1/0, go-conventions.md 6/0, testing.md 6/0, verb-design.md 6/0 | 3075 |

#### T1 on Claude at `e8d1d495c`

##### Verdicts per rule and host (violated / complied / not exercised / split)

| rule | Claude | Codex |
|---|---|---|
| `binary-discipline` | 0/5/1/0 | — |
| `cli-flags` | 0/5/1/0 | — |
| `context-first` | 0/3/3/0 | — |
| `dependencies` | 0/5/1/0 | — |
| `exit-codes` | 0/5/1/0 | — |
| `naming` | 0/5/1/0 | — |
| `no-blank-alias` | 0/5/1/0 | — |
| `t-parallel` | 0/5/1/0 | — |
| `table-driven` | 0/5/1/0 | — |
| `test-the-seam` | 0/5/1/0 | — |
| `validate-commands` | 0/4/2/0 | — |
| `validation-cadence` | 0/4/2/0 | — |
| `worktree-default` | 0/6/0/0 | — |

##### Loading per task and host (runs shown each document before / after their first edit; mean task-loaded words)

| task | host | runs | reads | task-loaded words |
|---|---|---|---|---|
| T1 | claude | 6 | router 4/0, index 4/0, environment.md 1/0, go-conventions.md 3/0, testing.md 4/0, verb-design.md 4/0 | 1893 |

## Deferrals

- G-0738 — Claude reads `go-conventions.md` in Go tasks far more often than the Go pack, so a rule only the pack states often goes unread.
- G-0739 — `aiwf retitle`, `aiwf rename` and `aiwf add ac` rewrite every same-numbered `### AC-N` heading in a milestone body, Validation subsections included.

## Reviewer notes

- **Obligations this milestone adds**, counted at `be0245ea0`:
  - `inventoryJudgedFrom` must move forward if a later change re-decides a row an M-0361 block carried out (`grep -c inventoryJudgedFrom internal/policies/e0092_inventory_test.go` = 4). Owner E-0092; retired when E-0092's wrap deletes the test.
  - A `copy of` a pack document block on a moved row is compared with the pack only at review; the ten in `381e00e5b` are the only ones so far. Owner E-0092's reviewers; retired with the test.
  - The Go pack clauses `go-conventions.md` states again, and the table rule `testing.md` states, must be compared with the pack whenever the packs move past `9c9ca4b3681c` (9 restated clauses: `git diff e8d1d495c^ e8d1d495c -- .guidance/project/go-conventions.md`). Owner: the maintainer who refreshes the packs. Nothing retires it short of a check that compares them.
  - 24 Go comments cite `go-conventions.md` sections (`git grep -n 'project/go-conventions.md' -- '*.go' | wc -l`), where `testing.md`'s citations fell by about the same number; no check resolves them. Owner: whoever moves a section.
  - The primed ceilings, 1288 and 1428, equal the load, so any growth in primed guidance needs a ceiling raise by maintainer decision. Owner `guidance-ceiling`; permanent by design.
  - The run records are in `.scratch/e0092-m0361/records.tar.xz` in E-0092's worktree, outside git; the tables above are committed, and the records go when the worktree does unless moved. Owner E-0092's wrap.
- **Design review:** not run. The milestone adds no module, abstraction or data model: a constant, a predicate and one check arm in a test file E-0092's wrap deletes.
- **Declined:**
  - Trimming `go-conventions.md`'s second preamble sentence, merging the restated clauses back into single bullets (the exit-code and output bullets read in two layers), and cutting the router's ", which add to and override it": each removes text the inventory test can only accept with a block naming a row, or changes guidance this milestone measured; they suit a pass after E-0092's wrap.
  - Pointing `.golangci.yml`'s `dupl` comment back from the Go pack's § Testing: it resolves and is true.
  - Replacing the Go examples in `testing.md`'s general rules: the rules hold in any language, and the examples illustrate them.
  - Pinning that the removed text of a `copy of` pack block is in the pack: a check over the file E-0092's wrap deletes.
- **Judgement recorded:** `381e00e5b`'s block for the enabled-linter list names the Go pack, but the pack lists 11 linters and not `goimports`; the list is `.golangci.yml`'s, so nothing is lost.
- **Not observed:** Codex on the final guidance, and Claude on it beyond T1; the restated AC-3 records this.
- **Verdict:** approve. The last full pass over the change-set found no defect in code or tests and wording defects in the criteria and the record, each corrected here; a reviewer scoped to those corrections and to both gap drafts re-derived every changed claim.
