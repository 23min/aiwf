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
table. The M-0357 AC-1 trial ran the Go row's coverage and clone-detector commands,
so a wrong name there goes red in a run; the mutation harness resolves on the path
but no trial ran it.

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
protected turns something red, or a guard is shown unreachable. It reports and
applies nothing. It needs no aiwf verb or configuration, and a per-stack table names the
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
nothing". The text under test is installed before the session starts, and each
run's transcript is checked for the text it loaded.

| Run | Input | Skill text | Result |
|---|---|---|---|
| Diff | `HEAD~1..HEAD` | `4626a2eef` | Every report part present, split by concern over three fresh agents |
| Unit | `internal/policies/guidance_ceiling.go` | `b362e9e20` | Every report part present |
| Prose | `HEAD~1..HEAD -- docs/` (3 files, +51 −5, no logic, no tests) | `1b09b9a56` | No logic or tests bucket stated; steps 3–9 skipped, each with its reason; nothing proposed |

**M-0357 AC-1.**
- Both reports hold the verdict counting cuts the scope blocked (diff 3; unit 4,
  one of them stopped by a constraint rather than a file),
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
- The prose run observed the shipped text and exercised the no-logic edge case.
  The unit run's text differs from it in the settling rule, which now names both
  routes — something red, or a guard shown unreachable — and in the skip, the
  outside-system disposition, the triage pointer and the harness and coverage
  fallbacks; its removals took those routes already. The diff run's text differs
  further in the Go row's clone-detector cell, and that run executed the command
  the shipped cell names.
- No run's transcript contains an `aiwf` command. The diff and unit runs read the
  live checkout of this repository read-only, as real data for differential tests
  and to rebuild the missing root files.

**M-0357 AC-2.** Every proposed removal carries one of the two forms the criterion
allows, and none rests on a green gate run:
- the demonstration that no caller reaches the protected state, or a measured
  equivalence, with the break that turns nothing red where one ran (diff run: 12
  code removals, e.g.
  `git -c diff.renames=copies log -z -M --name-status` → no `C` record, settling the
  unreachable copy arm);
- for a test, the break-to-tests table: every break turning it red turns another
  red (unit run: four link-form subtests, `L8: new reds:
  ['TestMeasureGuidanceLoad_ReferenceForms', 'TestRoutedDocuments']`).

**Against what M-0333 went on to change.** The hand-run trim `0f43b0b09` removed
four conditions: the router read's `rerr == nil`, the `ok` on an import read,
`target != ""` and `&& x.Name != name`. The diff run proposed the last two for
removal and found the first equivalent; the unit run found the second surviving
and declined to cut it. Both runs reported that the code counts `@` imports for
Codex, which Codex's documentation does not support — the rule `378a8cc41` changed.
The diff run also found a false `coverage:ignore`.

**Limits.** The `aiwf` binary stayed on the path, so the record shows it was not
called rather than that it could not be. Findings the runs make about the M-0333
code itself were not checked against the current tree.

**Gates at wrap**, on `milestone/M-0357-ship-the-on-demand-subtraction-skill`:
`make check-fast` → exit 0 (`golangci-lint run` → `0 issues.`, every package `ok`);
after the skill's last edit, `go test -count=1 ./internal/policies/
./internal/skills/` → `ok` and `-run TestClaudeArtifacts_MatchBaseline
./internal/cli/integration/` → `ok`; `aiwf check` → 0 errors.

## Deferrals

- (none)

## Reviewer notes

- The two settling routes for a removal — something goes red, or a guard is shown
  unreachable — are stated in the skill's prose, where D-0070 rules out a check;
  review is what holds them.
- Declined: the revert pointer to `wf-vacuity` stands although it is stricter than a
  separate working copy needs. Findings from obligations, the break-to-tests table
  and outside-system rules route to the report's existing sections rather than to
  new ones; the prose run asked the same for step 1's obligation differences.
- For M-0358: two parts of the milestone wrap's shape block are not in the skill —
  mandated comments, planning prose and tests pinning distinct rules do not count
  toward half, and the Deletions question. Replacing the block with a call to the
  skill drops both unless the wiring keeps them.
