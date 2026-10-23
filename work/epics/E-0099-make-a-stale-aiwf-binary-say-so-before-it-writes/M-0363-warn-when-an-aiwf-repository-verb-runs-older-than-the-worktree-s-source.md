---
id: M-0363
title: Warn when an aiwf-repository verb runs older than the worktree's source
status: in_progress
parent: E-0099
tdd: required
acs:
    - id: AC-1
      title: Doctor judges every binary that carries a commit
      status: met
      tdd_phase: done
    - id: AC-2
      title: Every verb in the aiwf repository warns on a stale binary
      status: met
      tdd_phase: done
    - id: AC-3
      title: make build leaves no bin/aiwf to reach for
      status: cancelled
      tdd_phase: done
---

## Goal

Working on aiwf with a binary built from different source than the worktree says so
before a verb runs, and `aiwf doctor` judges `make install` builds.

## Closes

- G-0471 — a verb run in the aiwf repository by a binary built from other source than
  the worktree's `HEAD` says so before it runs, except a plain `go build` or
  `go install` binary run in a linked worktree.

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

**Pass criterion**: `binaryStaleness` reads the revision a binary was built from — a
pseudo-version's hash, a Makefile stamp's `g<sha>`, a tagged release or a stamp at an
exact tag — and reports the binary as stale, with one message, when the source it was
built from differs from the ref's, and is silent when it matches; a `+dirty` or
`-dirty` build is judged by the commit it started from; `aiwf doctor`'s `binary:` row
shows it. **Edge cases**: a commit that changes only planning files or tests leaves the
binary current; a revision the repository lacks, `(devel)`, and a directory that is not
a Git repository are silent. **Code references**: revision extraction beside
`version.PseudoSHA` in `internal/version/version.go`; the table in
`internal/cli/doctor/binary_staleness_internal_test.go`.

### AC-2 — Every verb in the aiwf repository warns on a stale binary

**Pass criterion**: inside the aiwf repository, every verb run by a binary whose source
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

Cancelled: `make selfcheck` depends on the binary being named `aiwf`, because
`aiwf doctor --self-check` puts the binary's directory first on `PATH` for the hooks it
installs (`internal/cli/doctor/selfcheck.go`), and a `bin/aiwf` built from this
repository now warns once its source falls behind (AC-2). G-0654 stays open.

## Constraints

- Reuse `binaryStaleness` for both `aiwf doctor` and the pre-verb warning; no second
  staleness check.
- The warning never blocks a verb, and it is inert outside the aiwf repository.

## Design notes

- The check compares the source a binary was built from — `cmd`, `internal`, `go.mod`
  and `go.sum`, tests excluded — with `HEAD`'s, rather than its commit or its version: a
  commit that changes only planning files or tests leaves the binary running the same
  logic. Measured 2026-10-23: 261 of the last 300 first-parent commits on `main` changed
  nothing under those paths.
- The warning runs for every verb rather than only mutating ones: a stale read is acted
  on as current too, and one hook on the root command needs no classification of verbs.
- The check's cost grows with how far the binary's revision is from `HEAD`. Measured
  2026-10-23 in the E-0099 worktree (16 cores, load about 3.6), `aiwf version` run 20
  times per binary: about 4 ms per call for a build carrying no revision, 18–20 ms for
  one stamped at `HEAD`, about 72 ms for `v0.40.0` (870 commits behind) and about 94 ms
  for `v0.30.0` (4577 behind). `~/.claude/statusline.sh` never runs `aiwf`, so the
  warning cannot reach the statusline.

## Out of scope

- Refusing to run a stale binary, and rebuilding from source in the Git hooks.

## Dependencies

- None.

## References

- G-0471
- G-0654
- E-0099

## Release note

Inside the aiwf repository, every `aiwf` verb run by a binary whose source differs from
the worktree's `HEAD` prints `aiwf: warning: stale binary — built from <revision>, whose
source differs from HEAD` to stderr before it runs. Its output and exit status are
unchanged, a commit that touches only planning files or tests does not trigger it, and
a consumer repository sees nothing. A plain `go build` or `go install` binary run in a
linked worktree is not judged, because the go1.25 toolchain stamps it with the main
checkout's commit; build with `make diag-aiwf` there. `aiwf doctor`'s
`binary:` row judges `make install` builds, tagged releases and dirty builds by their
source instead of skipping them.

## Decisions made during implementation

- A Go pseudo-version is not judged inside a linked worktree: under go1.25.12, the
  devcontainer's `GOTOOLCHAIN`, `go build` there stamps the main checkout's commit.
  Measured 2026-10-22: a bare `go build` in the E-0099 worktree reported `e8987c72ca1f`,
  the main checkout's `HEAD`, while the worktree's was `fb4074d7bb11`. Measured
  2026-10-23: go1.27.1 stamps the same build with the worktree's own commit
  (`1db49ffee37b`), so a toolchain that does so retires the skip.

## Validation

`make ci` on the milestone branch with the code of `f1a1bac85`, 2026-10-23, go1.25.12 linux/amd64 in the devcontainer: exit 0.
`golangci-lint run` reported `0 issues`; every test package passed under `-race` with
coverage, including the diff-scoped `TestPolicy_BranchCoverageAudit`; the self-check
printed `self-check passed (29 steps).` `aiwf check --since` the epic branch, run by a
binary built from this branch, reported no error, the instruction-file fence included.

## Deferrals

- (none)

## Reviewer notes

- Obligations this adds, none of them mechanical (the policy and chokepoint counts are
  unchanged): `binarySource` in `binary_staleness.go` must keep covering every path the
  binary is built from, owned by whoever adds a source directory outside `cmd` and
  `internal`; the linked-worktree skip goes when the toolchain stamps worktree builds
  with their own commit, as go1.27.1 does, owned by the Go toolchain bump; and no
  subcommand may define its own `PersistentPreRun`, since cobra runs only the nearest
  one — unowned, retired by `cobra.EnableTraverseRunHooks` if a subcommand ever needs
  one.
- A revision the repository lacks is silent: its source cannot be compared.
- Doctor's `binary:` row has no linked-worktree skip, so for a plain go1.25 build in a
  nested worktree it judges the main checkout's commit against the trunk; it can then
  report stale against the wrong source when the main checkout lags the trunk. The row
  is opt-in and already judged clean bare builds that way; not measured.
- The pre-existing `expectedModule == ""` guard is redundant with the module-path check
  except for a `go.mod` with no `module` line; it is kept.
- Declined, each needing a case before code: honouring a verb's `--root` (the hook reads
  the working directory's repository); a guard against a subcommand overriding the
  hook; an in-process test of doctor's row call site, which `staleRowSuffix` pins; one
  ref for doctor's row and the warning (trunk and `HEAD` answer different questions);
  gating the linked-worktree skip on `runtime.Version()`; inlining `inLinkedWorktree`;
  removing `gitops.ShortSHA`, which other packages' tests call.
- The warning names no remedy: `make install` would install unmerged code from a
  milestone worktree and downgrade the binary when it is newer than the worktree.
