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
    - id: AC-2
      title: A mutating verb on a stale aiwf-repository binary warns first
      status: open
    - id: AC-3
      title: make build leaves no bin/aiwf to reach for
      status: open
---

## Goal

Working on aiwf with a binary built from different source than the worktree says so
before a verb writes, `aiwf doctor` judges `make install` builds, and `make build` stops
leaving a `bin/aiwf` that gets run instead of the release.

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

**Pass criterion**: `binaryStaleness` given a Makefile stamp whose commit differs from
the ref reports it as stale, with the same message a pseudo-version gets, and is silent
when the commit equals the ref; `aiwf doctor`'s `binary:` row shows it. **Edge cases**: a
`-dirty` stamp is skipped as a `+dirty` version is; a stamp at an exact tag with no
commit hash (`main@v0.40.0`) is skipped; describe's abbreviated hash is compared as a
commit prefix. **Code references**: commit extraction beside `version.PseudoSHA` in
`internal/version/version.go`; the shape table in
`internal/cli/doctor/binary_staleness_test.go`.

### AC-2 — A mutating verb on a stale aiwf-repository binary warns first

**Pass criterion**: inside the aiwf repository, a verb that writes the tree or commits,
run by a binary whose commit differs from the worktree's `HEAD`, prints
`binaryStaleness`'s warning (called with `HEAD`) to stderr before it writes, and its
result and exit status are otherwise unchanged. **Edge cases**: a read-only verb prints
nothing; a consumer repository prints nothing (the module-path check); a `+dirty` or
`-dirty` binary, a directory that is not a Git repository, and a binary at `HEAD` print
nothing. **Code references**: the warning is wired once for the mutating verbs, reusing
an existing classification of them where one exists; `binaryStaleness` itself is
unchanged apart from AC-1.

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
- The cost of the check before each mutating verb is measured at start (E-0099's open
  question).

## Out of scope

- Refusing to run a stale binary, and rebuilding from source in the Git hooks.

## Dependencies

- None.

## References

- G-0471
- G-0654
- E-0099
