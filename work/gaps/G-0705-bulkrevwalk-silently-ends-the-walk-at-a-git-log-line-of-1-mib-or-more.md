---
id: G-0705
title: BulkRevwalk silently ends the walk at a git log line of 1 MiB or more
status: open
priority: low
---
## What's missing

`splitOnMarker` (`internal/gitops/revwalk.go:241`) splits the `git log` output of
`BulkRevwalk` into per-commit records through a `bufio.Scanner` whose line ceiling
is 1 MiB, and never consults `scanner.Err()`. A line of 1 MiB or more ends the
scan, and the records already collected are returned as if they were the whole
history, with a nil error.

The format prints each extracted trailer value (the keys in `bulkTrailerKeys`,
`revwalk.go:110`) on a line of its own, with `unfold=true` joining a folded value
into one line. So a commit carrying one of those trailers with a value near 1 MiB —
`aiwf-reason:` and `aiwf-force:` are free text — is dropped together with every
commit `git log` prints after it. A `--raw` path line past the ceiling, which only
plumbing can build, keeps its commit with a truncated path list and drops the older
commits. No aiwf verb writes such a trailer on Linux, since `--reason` travels as
one argv string and an argument of 131,072 bytes fails with E2BIG; the trigger is a
commit message written outside the verbs.

Measured on Linux (go 1.25.11, git 2.54.0) against source at `adc2e6001`, in a
scratch repository of four commits where the third carries a 2 MiB `aiwf-reason:`
trailer. A throwaway `main` called `gitops.BulkRevwalk` and counted the records its
callback received:

    $ git rev-list --all --count
    4
    $ go run ./cmd/zzprobe <repo>
    record b7ca81803
    records=1 err=<nil>

Expected four records, or an error. Observed one record and a nil error. The same
probe returns all four records when the longest line is 1,048,575 bytes, and one
record at 1,048,576.

## Why it matters

`BulkRevwalk`'s one production caller is the history walk behind
`fsm-history-consistent` (`internal/check/fsm_history_walker.go:192`), which feeds
its `illegal-transition`, `forced-untrailered` and `manual-edit` findings. A
truncated walk sees no commit at or older than the long line, so those findings
vanish and `aiwf check` passes the push.

Measured end to end with an `aiwf` binary built from `adc2e6001`, in a scratch
consumer repo holding a hand-committed `wontfix → open` transition on a gap:
`aiwf check` exits 1 with `fsm-history-consistent illegal-transition`. After a newer
commit carrying a 2 MiB `aiwf-reason:` trailer it exits 0 and reports only a
provenance warning. The same history with a 1 KiB trailer still exits 1 with the
finding.

The walk turns a failed blob read, or an error returned by `BulkRevwalk`, into a
`history-walk-error` finding so that missing history is visible
(`internal/check/fsm_history_consistent.go:175`); this path loses history without
returning an error, so no finding marks it.
