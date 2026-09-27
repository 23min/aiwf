---
id: M-0357
title: Ship the on-demand subtraction skill
status: in_progress
parent: E-0097
tdd: advisory
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
demonstration that none can rather than a red run; a stack with no mutation harness,
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
table, and M-0357 AC-1's trial runs each tool the Go row names, so a wrong name goes
red there.

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
settles every removal it proposes by breaking what the removed thing protected. It
reports and applies nothing. It needs no aiwf, and a per-stack table names the
mutation harness, coverage profile and clone detector for Go, Python,
JavaScript/TypeScript and the JVM.

## Decisions made during implementation

- (none)

## Validation

Observed 2026-09-27 in the Linux development container: Claude Code 2.1.283,
`claude-opus-5-5`, Go 1.25.11, `golangci-lint` 2.12.2, `gremlins` on the path.

**Repository.** A two-commit clone of this repository's M-0333 change, built in the
scratchpad with bash (zsh aborts the whole `rm` on an unmatched glob):

```bash
git init -b main
git -C <aiwf> archive 29a73ee62 | tar -x   # then remove work .claude .agents aiwf.yaml CLAUDE.md AGENTS.md ROADMAP.md STATUS.md
git add -A && git commit -m base
git rm -rq . && git -C <aiwf> archive 0f43b0b09^ | tar -x   # same removal
git add -A && git commit -m "fence project guidance while preserving managed updates"
# install every wf-rituals SKILL.md into .claude/skills/, git-excluded
```

`29a73ee62` is M-0333's start and `0f43b0b09^` its state before the hand-run trim
landed, so both commits carry no aiwf configuration, planning tree, host artefacts
or hooks. The diff is 31 files, +3009 −52. With no root instruction file, 57 tests
fail at HEAD (51 in `internal/policies`), listed to each session as pre-existing.

**Method.** Each run is a fresh interactive session started in the clone, sent one
prompt naming the skill, the input, the M-0333 ticket (its Goal through
Dependencies sections), the pre-existing failures, and a report path, with "apply
nothing". A session reads the skill text when it starts, so the text under test is
installed before the session is started, and each run is confirmed against its
transcript.

| Run | Input | Skill text | Result |
|---|---|---|---|
| Diff | `HEAD~1..HEAD` | `4626a2eef` | Every report part present, split by concern over three fresh agents |
| Unit | `internal/policies/guidance_ceiling.go` | `b362e9e20` | Every report part present |

**M-0357 AC-1.**
- Both reports hold the verdict counting cuts the scope blocked (diff 3, unit 4),
  behaviour changes with the records that state the old behaviour, the guard table
  with a break per guard, and the handoffs to `wf-vacuity`, `wf-rethink` and
  `wf-structural-sweep`. The unit run proposes no behaviour change and names the
  records for the two it assessed and declined. Each skipped step is stated with
  its reason.
- Tools from the Go row: the coverage profile ran in both
  (`go test -coverprofile`, `go tool cover -func`). The clone detector ran in both
  as `golangci-lint run --enable-only dupl ./internal/policies/` → `0 issues`,
  each noting the configuration excludes it from test files. `gremlins` ran in
  neither: the unit run skipped it because the package has failing tests, the diff
  run on cost; both used the manual probe and said so.
- The diff run observed `4626a2eef`, which differs from the shipped text only in
  the Go row's clone-detector cell; that run executed the command the shipped cell
  names.
- Neither transcript contains an `aiwf` command. Both read the live checkout of
  this repository read-only, as real data for differential tests and to rebuild
  the missing root files.

**M-0357 AC-2.** Every proposed removal carries one of the two forms the criterion
allows, and none rests on a green gate run:
- a break that turns nothing red, with the demonstration that no caller reaches the
  protected state, or a measured equivalence (diff run: 12 code removals, e.g.
  `git -c diff.renames=copies log -z -M --name-status` → no `C` record, settling the
  unreachable copy arm);
- for a test, the break-to-tests table: every break turning it red turns another
  red (unit run: four link-form subtests, `L8: new reds:
  ['TestMeasureGuidanceLoad_ReferenceForms', 'TestRoutedDocuments']`).

**Against the hand-run trim of the same state**, recorded in the initiative doc:
both runs found the managed-block extraction duplicate and the reuse blocked by
scope, the guards no test pins, the equivalent `err != nil` operands, and the rule
the code applies to Codex that Codex's own documentation does not state — the
change M-0333 went on to make. The diff run also found a false `coverage:ignore`.

**Limits.** The `aiwf` binary stayed on the path, so the record shows it was not
called rather than that it could not be. Findings the runs make about the M-0333
code itself were not checked against the current tree.

**Gates at wrap**, on `milestone/M-0357-ship-the-on-demand-subtraction-skill`:
`make check-fast` → exit 0 (`golangci-lint run` → `0 issues.`, every package `ok`);
`aiwf check` → 0 errors.

## Deferrals

- (none)

## Reviewer notes

- (none)
