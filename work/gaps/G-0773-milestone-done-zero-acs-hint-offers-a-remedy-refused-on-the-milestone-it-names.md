---
id: G-0773
title: milestone-done-zero-acs hint offers a remedy refused on the milestone it names
status: open
---
## What's missing

The hint for `milestone-done-zero-acs`, in `hintTable` in `internal/check/hint.go` (line 179),
offers a remedy the kernel refuses on the milestone the finding names: "add one via
`aiwf add ac <milestone-id> --title "..."` first if this was unintentional".
`milestoneDoneZeroACs` in `internal/check/acs.go` reports only a non-archived milestone with
`status: done`, and on that milestone `aiwf add ac` is refused: the new AC would be `open` on a
`done` milestone, which the `milestone-done-incomplete-acs` rule forbids ("Milestone-done
implies AC progress" in `docs/design/design-decisions.md`, line 144). `aiwf add ac` takes no
`--force`.

Measured on Linux 7.0.14 in the devcontainer, with `aiwf` built from a `git archive` of main at
bd818e14d, in a scratch repository: E-0001 `active`, M-0001 `draft` with `tdd: none` and no AC,
on the epic branch. E-0001 and M-0001 here are the scratch repository's own; in this tree those
ids name unrelated archived entities.

```
$ aiwf promote M-0001 in_progress --force --reason "probe"
aiwf promote M-0001 draft -> in_progress
exit 0
$ aiwf promote M-0001 done
aiwf promote M-0001 in_progress -> done
exit 0
$ aiwf check --verbose | grep milestone-done-zero-acs
work/epics/E-0001-pilot-epic/M-0001-pilot-milestone.md:1: warning milestone-done-zero-acs: milestone M-0001 is done with zero acceptance criteria — hint: advisory only — a done milestone with no acceptance criteria is a legitimate end state; add one via `aiwf add ac <milestone-id> --title "..."` first if this was unintentional
$ echo "The thing holds." | aiwf add ac M-0001 --title "Something holds" --body-file -
work/epics/E-0001-pilot-epic/M-0001-pilot-milestone.md:4: error milestone-done-incomplete-acs: milestone M-0001 is done but 1 AC(s) still open: AC-1 — hint: promote each open AC via `aiwf promote <milestone-id>/AC-N <met|deferred|cancelled>`, or override with `aiwf promote <milestone-id> done --force --reason "..."` (the standing check still surfaces this). `--force` overrides only the precondition this finding names, not the finding itself: it is sovereign, so it requires a `human/...` actor, and every other check still runs.
work/epics/E-0001-pilot-epic/M-0001-pilot-milestone.md:1: warning acs-body-coherence/missing-heading: M-0001/AC-1 has no `### AC-1` heading in the milestone body — hint: add a `### AC-<N> — <title>` heading in the milestone body via `aiwf edit-body <milestone-id>`, or drop the AC from the `acs:` frontmatter

2 findings (1 errors, 1 warnings)
exit 1
$ aiwf promote M-0001/AC-1 met
aiwf promote: M-0001 has no AC-1 in acs[]
exit 2
```

`HEAD` was the same commit before and after the `aiwf add ac`, and the milestone's frontmatter
still had no `acs:` key.

## Why it matters

An operator who sees the warning and follows its hint is refused. The refusal reports AC-1 as
open, an AC the refused command never wrote, so promoting it as the refusal's hint directs fails
too. The hint names no route that works for the milestone it reports.
