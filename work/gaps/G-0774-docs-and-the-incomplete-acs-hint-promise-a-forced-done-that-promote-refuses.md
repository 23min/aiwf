---
id: G-0774
title: Docs and the incomplete-ACs hint promise a forced done that promote refuses
status: open
---
## What's missing

D-0110 settles that `--force` does not take a milestone to `done` while an AC is open, and
`Promote` in `internal/verb/promote.go` refuses it: its projection check runs under `--force`
too and reports `milestone-done-incomplete-acs` as an error. These sources say `--force` takes
the milestone there:

- the "Milestone-done implies AC progress" bullet in `docs/design/design-decisions.md` (line
  144): "`--force --reason` overrides the verb-time refusal; the standing check still reports";
- the `milestone-done-incomplete-acs` hint in `hintTable` in `internal/check/hint.go` (line
  177): "or override with `aiwf promote <milestone-id> done --force --reason "..."`";
- `internal/skills/embedded/aiwf-promote/SKILL.md` (line 73): "`--force` lets the milestone
  reach `done`";
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`
  (line 274): "`--force` lands the verb but leaves the standing check red";
- the comment on `milestoneDoneIncompleteACs` in `internal/check/acs.go` (lines 346–348): "a
  milestone that became `done` via --force --reason while ACs were still open keeps surfacing
  the inconsistency".

`forceOfferingHintKeys` in `internal/check/hint_force_caveat_test.go` lists the hint among those
whose remedy an operator can act on by typing `--force`, so the test fails if the hint stops
offering it. D-0109's reasoning and G-0575's account of the hint describe these sources as they
stand. The section of `docs/audits/gap-truth-audit.md` at line 213 recorded the same refusal for
the skill and the hint. G-0575 measured the hint's `--force` remedy on a milestone already
`done`, where it converges to a NoOp.

Measured on Linux 7.0.14 in the devcontainer, with `aiwf` built from a `git archive` of main at
bd818e14d (no Go change since, up to ce4e6d89c), in a scratch repository: E-0001 `active`,
M-0001 `draft` with `tdd: none` and AC-1 `open` with body prose, on the epic branch. E-0001 and
M-0001 here are the scratch repository's own; in this tree those ids name unrelated archived
entities.

```
$ aiwf promote M-0001 in_progress
aiwf promote M-0001 draft -> in_progress
exit 0
$ aiwf promote M-0001 done --force --reason "probe"
work/epics/E-0001-pilot-epic/M-0001-pilot-milestone.md:4: error milestone-done-incomplete-acs: milestone M-0001 is done but 1 AC(s) still open: AC-1 — hint: promote each open AC via `aiwf promote <milestone-id>/AC-N <met|deferred|cancelled>`, or override with `aiwf promote <milestone-id> done --force --reason "..."` (the standing check still surfaces this). `--force` overrides only the precondition this finding names, not the finding itself: it is sovereign, so it requires a `human/...` actor, and every other check still runs.
work/epics/E-0001-pilot-epic/M-0001-pilot-milestone.md:1: warning milestone-done-empty-release-note: milestone M-0001 is done without a `## Release note` an author wrote; the epic wrap composes its changelog entry from these notes, so this milestone's change reaches the release described by nobody who did the work — write the user-visible delta, or the words "no user-visible change" when there is none — hint: write the milestone's user-visible delta under the release-note heading — what a consumer can now do, or what changed under them, or the words "no user-visible change" when there is none — then commit it with `aiwf edit-body <milestone-id>`

2 findings (1 errors, 1 warnings)
exit 1
```

The unforced `aiwf promote M-0001 done` before it printed the same two findings, exit 1.
`HEAD` was the same commit before and after both, the milestone stayed `in_progress` and AC-1
stayed `open`.

## Why it matters

The finding's hint offers two remedies, and the same finding refuses the second, so an operator
who takes it is refused by the error it offered to override.

The `aiwf-promote` skill is what an assistant reads when it promotes a milestone, and
`docs/design/` is in the Normative tier, read as current truth. Both present `--force` as a way
to finish a milestone with open ACs, so a plan built on either fails at the verb.
