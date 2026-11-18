---
id: G-0772
title: Spec table's milestone start cell declares none of the refusals promote enforces
status: open
---
## What's missing

Besides the branch guard (`requireExpectedBranchForActivatingTransition`), which the generated
reference leaves out as branch choreography, `Promote` in `internal/verb/promote.go` refuses a
milestone's `draft → in_progress` on two AC preconditions: no AC at all
(`requireNonEmptyACsAtMilestoneStart`, D-0039) and an AC with no prose under its `### AC-N`
heading (`requireNonEmptyACBodiesAtMilestoneStart`, G-0216). The workflow spec table declares
neither. The milestone `draft → in_progress` cell in `milestoneRules()` in
`internal/workflows/spec/rules.go` is a single Legal row with no precondition and no Illegal
row, while the table declares other refusals as Illegal rows, as it does for
`milestone-done-incomplete-acs` on `in_progress → done` in the same function (line 402). The
generated `docs/reference/workflow-legality.md` (line 75) renders the start as one Legal row
with blank conditions, and its global restrictions list neither refusal.

The positive-cell test driver works around the omission: the comment on
`seedACIfMilestoneActivating` in `internal/policies/m0124_positive_driver_test.go` (line 322)
says an empty `acs[]` (or an AC with no body) "would now collide with a real (non-forced)
verb-time refusal", so the driver seeds an AC with body prose.

ANTI-0001, tracked by G-0771, states the zero-AC case as a pattern the kernel does not
police.

Measured in this tree at bd818e14d:

```
$ grep -n 'milestone | draft | promote | in\\_progress' docs/reference/workflow-legality.md
75:| milestone | draft | promote | in\_progress | Legal |  | none | false |  |
```

And on Linux 7.0.14 in the devcontainer, with `aiwf` built from a `git archive` of main at
bd818e14d, in a scratch repository: E-0001 `active`, M-0001 `draft` with `tdd: none`, on the
epic branch. E-0001 and M-0001 here are the scratch repository's own; in this tree those ids
name unrelated archived entities. With no AC:

```
$ aiwf promote M-0001 in_progress
aiwf promote: cannot promote M-0001 to "in_progress": milestone has no acceptance criteria; add one with `aiwf add ac M-0001 --title "..."` first, or pass --force to override
exit 2
```

On a fresh copy, after `aiwf add ac M-0001 --title "Something holds"` (no body, exit 0):

```
$ aiwf promote M-0001 in_progress
aiwf promote: cannot promote M-0001 to "in_progress": M-0001/AC-1 has no body content; write prose under its `### AC-1` heading first
exit 2
```

## Why it matters

D-0101 publishes the generated reference as the legality reference and keeps predicates visible
because "presenting only origin, target and a green/red outcome would turn conditional
statements into unconditional advice". The start row is that case: a request the verb refuses
is shown as Legal with no condition.
