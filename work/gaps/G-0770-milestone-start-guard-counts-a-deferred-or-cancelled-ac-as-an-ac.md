---
id: G-0770
title: Milestone start guard counts a deferred or cancelled AC as an AC
status: open
---
## What's missing

D-0109 refines D-0039's start refusal: `draft → in_progress` needs at least one AC that is
`open` or `met`. The kernel counts entries in `acs[]` instead, and a `deferred` or `cancelled` AC
stays there, so a milestone whose every AC is deferred or cancelled passes:

- `requireNonEmptyACsAtMilestoneStart` in `internal/verb/promote.go` refuses only when
  `len(e.ACs) == 0`, so the milestone starts without `--force`;
- the draft-stage `zero-acs` warning and `milestone-done-zero-acs` in `internal/check/acs.go`
  test `len(e.ACs)` the same way, so neither fires.

Measured on Linux 7.0.14 in the devcontainer, with `aiwf` built from a `git archive` of main at
bd818e14d, in a scratch repository: E-0001 `active`, M-0001 `draft` with `tdd: none` and AC-1
`open`, on the epic branch. E-0001, M-0001 and M-0002 here are the scratch repository's own; in
this tree those ids name unrelated archived entities.

```
$ aiwf cancel M-0001/AC-1
aiwf cancel M-0001/AC-1 -> cancelled
exit 0
$ aiwf promote M-0001 in_progress
aiwf promote M-0001 draft -> in_progress
exit 0
$ aiwf promote M-0001 done
aiwf promote M-0001 in_progress -> done
exit 0
$ grep -E "^status:|^      status:" work/epics/*/M-0001-*.md
status: done
      status: cancelled
$ aiwf check | grep -c zero-acs
0
```

On a fresh copy, `aiwf promote M-0001/AC-1 deferred` (`open -> deferred`, exit 0) followed by
`aiwf promote M-0001 in_progress` printed `aiwf promote M-0001 draft -> in_progress`, exit 0. On
another, `aiwf cancel M-0001/AC-1` alone, with the milestone still `draft`, left
`aiwf check | grep -c milestone-draft-incomplete-acs` at `0`. For comparison, a milestone with
no AC at all, started with `--force` and promoted to `done`, makes `aiwf check` print
`milestone-done-zero-acs (warning) × 1 — milestone M-0002 is done with zero acceptance
criteria`.

## Why it matters

The refusal D-0039 put on starting a milestone with nothing to deliver is bypassed by adding an
AC and cancelling or deferring it. The milestone starts without the `--force --reason` that
D-0039 requires, so no reason is recorded, and it can then reach `done` while the warnings that
make a milestone without ACs visible stay silent, so the state is reported nowhere.
