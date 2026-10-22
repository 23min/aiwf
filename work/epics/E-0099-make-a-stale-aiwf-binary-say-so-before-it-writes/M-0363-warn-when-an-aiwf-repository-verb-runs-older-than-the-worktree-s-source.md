---
id: M-0363
title: Warn when an aiwf-repository verb runs older than the worktree's source
status: in_progress
parent: E-0099
tdd: required
acs:
    - id: AC-1
      title: Doctor judges every binary that carries a commit
      status: open
      tdd_phase: red
    - id: AC-2
      title: Every verb in the aiwf repository warns on a stale binary
      status: open
    - id: AC-3
      title: make build leaves no bin/aiwf to reach for
      status: open
---

## Goal

Working on aiwf with a binary built from different source than the worktree says so
before a verb runs, `aiwf doctor` judges `make install` builds, and `make build` stops
leaving a `bin/aiwf` that gets run instead of the release.

## Closes

- G-0471 — a verb run in the aiwf repository by a binary built from other source than
  the worktree's `HEAD` says so before it runs.
- G-0654 — `make build` no longer leaves a `bin/aiwf` to reach for.

## Context

`binaryStaleness` (`internal/cli/doctor/binary_staleness.go`) takes the ref to compare
with, reads the binary's commit from a Go pseudo-version through `version.PseudoSHA`,
and is silent outside the aiwf module. `aiwf doctor` calls it with the trunk ref. A
`make install` stamp, `<branch>@<git describe --always --dirty>`, carries its commit as
`g<sha>` (or as the whole describe output in a clone with no tags) and is skipped today
(G-0471). `make build` writes `bin/aiwf` (`Makefile`, `build:`), which only
`make selfcheck` consumes (G-0654).

## Acceptance criteria

### AC-1 — Doctor judges every binary that carries a commit

**Pass criterion**: `binaryStaleness` reads the binary's commit from a pseudo-version,
from a Makefile stamp's `g<sha>`, and from a tagged release or a stamp at an exact tag
through that tag in the repository, reports a commit that differs from the ref as stale
with one message, and is silent when it equals the ref; a `+dirty` or `-dirty` build is
judged by the commit it was built from; `aiwf doctor`'s `binary:` row shows it.
**Edge cases**: a tag the repository lacks, `(devel)`, and a directory that is not a
Git repository are silent; describe's abbreviated hash is compared as a commit prefix.
**Code references**: commit extraction beside `version.PseudoSHA` in
`internal/version/version.go`; the shape table in
`internal/cli/doctor/binary_staleness_test.go`.

### AC-2 — Every verb in the aiwf repository warns on a stale binary

**Pass criterion**: inside the aiwf repository, every verb run by a binary whose commit
differs from the worktree's `HEAD` prints `binaryStaleness`'s warning (called with
`HEAD`) to stderr before it runs, and its stdout, result and exit status are unchanged.
**Edge cases**: a consumer repository prints nothing (the module-path check); a binary
at `HEAD`, `(devel)`, and a directory that is not a Git repository print nothing.
**Code references**: one hook on the root command; `binaryStaleness` unchanged apart
from AC-1.

### AC-3 — make build leaves no bin/aiwf to reach for

**Pass criterion**: after `make build`, no executable named `aiwf` exists under `bin/`,
and `make selfcheck` still passes using the binary `make build` produces. **Edge
cases**: `make diag-aiwf`'s `bin/aiwf-diag` is unchanged; `make install` is unchanged.
**Code references**: `Makefile` (`build`, `selfcheck`); a recipe check through the
`makeDryRun` helper in `internal/policies/`, as `makefile_version_stamp_test.go` does.

## Constraints

- Reuse `binaryStaleness` for both `aiwf doctor` and the pre-verb warning; no second
  staleness check.
- The warning never blocks a verb, and it is inert outside the aiwf repository.

## Design notes

- Comparing for a different commit rather than an older one reuses `binaryStaleness`
  as it is; a binary built from different source runs different logic either way.
- The warning runs for every verb rather than only mutating ones: a stale read is acted
  on as current too, and one hook on the root command needs no classification of verbs.
- Measured 2026-10-22 in the devcontainer: two `git rev-parse` calls cost about 23 ms per
  verb, against about 6 ms for `aiwf version` itself, paid only inside the aiwf
  repository. `~/.claude/statusline.sh` never runs `aiwf`, so the warning cannot reach
  the statusline.
- `.guidance/project/environment.md` states that nothing mechanically blocks a
  stale-PATH call; closing G-0471 corrects that sentence, under the instruction-file
  fence.

## Out of scope

- Refusing to run a stale binary, and rebuilding from source in the Git hooks.

## Dependencies

- None.

## References

- G-0471
- G-0654
- E-0099
