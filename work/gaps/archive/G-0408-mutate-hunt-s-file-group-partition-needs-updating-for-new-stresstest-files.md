---
id: G-0408
title: mutate-hunt's file-group partition needs updating for new stresstest files
status: addressed
addressed_by_commit:
    - 7f6dda3a
---
## What's missing

`mutate-hunt.yml`'s documented four-group file split for scoped
`internal/stresstest` dispatches (introduced by G-0407's part 2) has no
mechanism to stay in sync when the package gains a new production file. A
file added after the split was authored — `promote_on_wrong_branch_detection.go`,
landed via G-0270's merge — matches none of the four exclude patterns, so it
falls through every dispatch's exclusion and gets mutated redundantly in all
four instead of exactly one.

## Why it matters

Confirmed empirically: a 4-way scoped-dispatch validation run against
`internal/stresstest` found this file's 9 mutants repeated identically
across all four dispatch logs, inflating each run's mutant count and CI
time, and making the workflow's own "these four groups partition ... with
no overlap" documentation claim false the moment a new file lands. Left
unaddressed, every future file added to the package silently drifts the
partition further out of sync, with nothing but a stale comment to notice.