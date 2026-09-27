---
id: M-0357
title: Ship the on-demand subtraction skill
status: in_progress
parent: E-0097
tdd: none
acs:
    - id: AC-1
      title: The skill runs on a diff and on a named unit without aiwf, reporting every part
      status: met
    - id: AC-2
      title: Every proposed removal is settled by a command that goes red
      status: met
    - id: AC-3
      title: Each tool the per-stack table names for this stack resolves
      status: cancelled
---

## Goal

Ship a stack-neutral skill that asks of a diff or a named unit whether the change
needs everything it adds, and settles every removal it proposes by a command rather
than by reading.

## Closes

- (none)

## Context

The procedure exists today only as an inline block in the milestone wrap, where the
patch ritual and consumer repositories cannot reach it. The skills around it each
default to adding, strengthening or keeping; none defaults to removal, and the code
review checklist explicitly guards against it. The design, and two hand-run trials
with what each got wrong, are in
[`subtraction-review-on-demand.md`](../../../docs/initiatives/archive/subtraction-review-on-demand.md).

## Acceptance criteria

### AC-1 — The skill runs on a diff and on a named unit without aiwf, reporting every part

The skill runs twice in a repository that does not use aiwf — once over a diff, once
over a named unit — and each run returns a report holding the verdict that counts
cuts the scope blocked, the behaviour changes awaiting approval with the records
that state the old behaviour, the keep-or-remove guard table with per-guard
evidence, and the handoffs to the test-sufficiency, design and whole-codebase
lenses. **Pass criterion**: a recorded trial per run naming the repository, the
input and the command, where each part of the report is present or explicitly
stated absent with its reason, and each tool the run takes from the per-stack table
is shown running. The repository is a scratch clone carrying no aiwf configuration,
planning tree, host artefacts or hooks, with only the generic workflow skills installed.
**Edge cases**: a diff with no logic bucket at all, which produces a stated skip
rather than an invented finding; a diff where the scope blocks a cut, which the
verdict counts rather than omits; the aiwf binary stays on the path, so the record
shows it was not called rather than that it could not be. **Code references**: the
skill body under the embedded ritual tree; the trial records land in this spec's
`## Validation`.

### AC-2 — Every proposed removal is settled by a command that goes red

Each removal the skill proposes carries a command whose output shows something going
red when what the removed thing protected is broken. **Pass criterion**: in both
recorded runs, every proposed removal carries its command and that command's
output; a proposal resting on a green gate run fails this criterion. **Edge cases**:
a removal whose protected state no caller can produce, where the evidence is the
demonstration that none can rather than a red run; a cut that changes no result,
which is a rewrite settled by a differential test; a stack with no mutation harness,
where the manual probe stands in and the report says so. **Code references**: the
break step in the skill body; G-0660 records the oracle this follows.

### AC-3 — Each tool the per-stack table names for this stack resolves

Every tool the per-stack table names for this repository's stack exists. **Pass
criterion**: a check resolving each command the table names for Go — the mutation
harness, the coverage profile and the clone detector — against this repository, and
failing when one names something absent. **Edge cases**: a tool that exists but is
switched off in configuration, which resolves and is still reported unavailable; a
stack row with no tool for a slot, which names its fallback rather than leaving the
slot blank. **Code references**: a new check under `internal/policies/`; the table in
the skill body.

**Cancelled.** No success criterion in the epic requires a standing check on the
table. The M-0357 AC-1 trial ran each of the Go row's three tools, so a wrong name
there fails in a run.

## Constraints

- The skill states each procedure once and points at the skills that own the others
  rather than restating them.
- Consumer-scoped: imperative instruction only — no aiwf id, no path into this tree,
  no development history and no rationale.
- Stack-neutral: every step named by its method, with a per-stack table naming the
  mutation harness, the coverage profile and the clone detector.
- No aiwf verb is required. Where aiwf is present, the gap and decision recorders are
  named as options, not dependencies.
- A question terminates on a disposition or on a command's result, never on a
  reading (G-0585).
- It advises and never blocks: no gate, no commit refused.

## Design notes

- [`subtraction-review-on-demand.md`](../../../docs/initiatives/archive/subtraction-review-on-demand.md)
  holds the twelve-step sketch and the defects both trials found in the trims
  themselves; build from it rather than re-deriving.
- D-0070 sets the evidence form: a relationship check where one exists, a recorded
  trial otherwise.
- **Retirement trigger**: the skill is retired if a re-run over recent patches and
  milestones finds nothing the reviews that already ran did not.

## Surfaces touched

- A new skill under `internal/skills/embedded-rituals/plugins/wf-rituals/skills/`

## Out of scope

- Wiring into the wrap rituals, and the skip threshold.
- Whole-codebase discovery, which `wf-structural-sweep` owns, and design rebuild,
  which `wf-rethink` owns; this skill recommends them.
- Applying any rewrite without approval.

## Dependencies

- (none)

## References

- [`docs/initiatives/archive/subtraction-review-on-demand.md`](../../../docs/initiatives/archive/subtraction-review-on-demand.md)
- D-0070 — the limits on pinning shipped prose.
- G-0585 — rituals that clear a question by reading it.
- G-0660 — the repaired oracle for a proposed removal.
- G-0533, G-0253 — the clone detector switched off over the test corpus, and
  statement-scoped coverage; both are conditions the skill's steps meet in this repo.

## Release note

New `wf-trim` skill in the generic workflow rituals: ask of one diff or one named
unit whether the change needs everything it adds. It looks for duplicated jobs,
logic that compresses, guards no caller reaches and tests no break needs, and it
settles every removal it proposes by a command: breaking what the removed thing
protected turns something red, or a guard is shown unreachable; a cut that
changes no result must pass a differential test. It reports, and applies nothing
until approved. It needs no aiwf verb or configuration, and a per-stack table names the
mutation harness, coverage profile and clone detector for Go, Python,
JavaScript/TypeScript and the JVM.

## Decisions made during implementation

- (none)

## Validation

Observed 2026-09-27 in the Linux development container: Claude Code 2.1.283,
`claude-opus-5-5`, Go 1.25.11, `golangci-lint` 2.12.2, `gremlins` on the path.

**Repository.** A two-commit clone of this repository's M-0333 change:

```bash
git init -b main
git -C <aiwf> archive 29a73ee62 | tar -x   # then remove work .claude .agents aiwf.yaml CLAUDE.md AGENTS.md ROADMAP.md STATUS.md
git add -A && git commit -m base
git rm -rq . && git -C <aiwf> archive 0f43b0b09^ | tar -x   # same removal
git add -A && git commit -m "fence project guidance while preserving managed updates"
# install every wf-rituals SKILL.md into .claude/skills/, git-excluded
```

`29a73ee62` is M-0333's start and `0f43b0b09^` its state before the hand-run trim
landed. Neither commit carries aiwf configuration, a planning tree, host artefacts
or hooks; `.guidance/` stays, as input data the code under test reads. The diff is
31 files, +3009 −52. With no root instruction file, 57 tests
fail at HEAD (51 in `internal/policies`), listed to each session as pre-existing.

**Method.** Each run is a fresh interactive session started in the clone, sent one
prompt naming the skill, the input, the M-0333 ticket (its Goal through
Dependencies sections), the pre-existing failures, and a report path, with "apply
nothing". Every run loaded the shipped text, `36e5c1494`, installed before the
session started; each transcript was checked for it.

| Run | Input | Verdict |
|---|---|---|
| Diff | `HEAD~1..HEAD` | 3 removals, 1 rewrite, 5 cuts blocked by scope |
| Unit | `internal/policies/guidance_ceiling.go` | 0 removals, 1 rewrite, 3 cuts blocked |
| Prose | `HEAD~1..HEAD -- docs/` (3 files, +51 −5) | 0 removals, 0 rewrites, 0 blocked; 1 cut for approval |

**M-0357 AC-1.**
- Every report holds the verdict counting blocked cuts, behaviour changes with the
  records that state the old behaviour, the guard table with a break per guard, and
  the handoffs; a section with nothing in it says so. Each skipped step is stated with its reason.
- Tools from the Go row. Coverage profile: the diff and unit runs
  (`go test -coverprofile`). Clone detector: both, noting that the configuration
  excludes test files — `golangci-lint run --enable-only dupl ./internal/policies/`
  → `0 issues` in the unit run; the diff run filtered the output of the same command
  over `./internal/policies/...` and printed nothing, and the unfiltered command in
  the clone → `0 issues.` Mutation harness: the diff run ran `gremlins`
  over the four new files with the failing tests skipped (154 killed, 6 lived, 8
  not covered, 1 timed out) and re-applied each mutant; the unit run skipped it
  because the package has failing tests, and broke conditions by hand.
- No-logic edge case: the prose run reported no logic and no tests bucket and
  skipped steps 3–5, 7, 8 and 9, each with its reason.
- No run called the `aiwf` on the path. The prose run built the clone's own
  `cmd/aiwf` and ran its `check` at both commits, as part of the code under review.
- The unit run read outside the clone, read-only: the live checkout's root
  instruction files, guidance and planning tree (the epic and the M-0333 and M-0335
  specs, cited as obligation sources), and a worktree holding the later M-0333
  commits, as real data for its differential test. Its Codex finding precedes every
  one of those reads.

**M-0357 AC-2.** No proposed removal rests on a green gate run. Each is settled by
a command and its output:
- dead: the copy-status arm of the log parser, shown unreachable because the one
  invocation passes `-M` and never `-C` — `git log -z -M --name-status` over a copy
  gives `M|a.md|A|b.md|` under every `diff.renames` setting — beside the break that
  leaves the package `ok`;
- red, for two tests: the break-to-tests table shows every break turning each red
  also turns others red (`breaks= 44 unique= 0`, `breaks= 43 unique= 0`);
- rewrites carry their differential test (diff run: 200,000 generated inputs and
  the real log, identical output; unit run: 30,000 generated trees and three real
  ones). Cuts that change no result without one are not proposed: the diff run
  withholds the `rev == ""` operand for want of a differential test, and the unit
  run keeps both `err != nil` operands and an import-read `ok`, whose outcome rests
  on undocumented behaviour.

**Limits.** The `aiwf` binary stayed on the path, so the record shows it was not
called rather than that it could not be. Findings the runs make about the M-0333
code itself were not checked against the current tree. Step 2 does not say
whether step 6 runs when a change has neither logic nor tests; the prose run
applied it to documents under a settling rule of its own and proposed removing one
for approval (G-0723).

**Gates at wrap**, on `milestone/M-0357-ship-the-on-demand-subtraction-skill`:
`make check-fast` → exit 0 (`golangci-lint run` → `0 issues.`, every package `ok`);
after the skill's last edit, `go test -count=1 ./internal/policies/
./internal/skills/` → `ok` and `-run TestClaudeArtifacts_MatchBaseline
./internal/cli/integration/` → `ok`; `aiwf check` → 0 errors.

## Deferrals

- G-0723 — `wf-trim` step 2 does not say whether step 6 runs on a change with
  neither logic nor tests.

## Reviewer notes

- How a removal is settled — something goes red, a guard is shown unreachable, or
  a cut that changes no result passes a differential test — and that a test written
  only to reach a dead guard goes with it, are stated in the skill's prose, where
  D-0070 rules out a check; review is what holds them.
- Declined: the revert pointer to `wf-vacuity` stands although it is stricter than a
  separate working copy needs. Findings from obligations, the break-to-tests table
  and outside-system rules route to the report's existing sections rather than to
  new ones; the prose run asked the same for step 1's obligation differences.
- M-0358 carries the parts of the milestone wrap's shape block the skill does not.
- Left for the next edit to the skill, since any edit moves the text the trial
  observed: step 5's "A caller can → the guard stays" reads against its own third
  sentence, which sends a guard whose removal changes no result to the rewrite
  step; the opening says the second agent confirms each removal, where step 11
  covers rewrites too; the dead-route constraint omits the break step 6 requires.
- E-0097's constraint names only the red route for a removal; the skill also
  settles a guard shown dead and a cut that changes no result. Reconcile at the
  epic wrap.
