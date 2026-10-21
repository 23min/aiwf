---
id: E-0099
title: Make a stale aiwf binary say so before it writes
status: proposed
---

## Goal

An aiwf command run by a binary older than the artifacts or source it acts on says
so before it writes, instead of silently rolling installed artifacts back or running
older logic.

## Context

`aiwf update` and `aiwf worktree add` both materialize artifacts through
`initrepo.RefreshArtifacts` (`internal/cli/update/update.go`,
`internal/cli/worktree/worktree.go`). Each artifact is written from the running
binary's embedded snapshot without consulting what is on disk, so a binary older than
the installed artifacts moves them backwards and reports `updated` (G-0600). The
guidance fragment (`aiwf-version:`) and the statusline script
(`# aiwf-statusline version:`) carry a version stamp; skills, templates and agent cards
carry none, so even a comparing guard has nothing to read for most families.

In the aiwf repository itself the `aiwf` on `PATH`, or the `bin/aiwf` that `make build`
leaves behind (G-0654), often predates the working tree, and every verb then runs older
logic with no signal (G-0471). `aiwf doctor`'s staleness row
(`internal/cli/doctor/binary_staleness.go`) skips tagged releases and every binary whose
version is not a Go pseudo-version, which includes every `make install` build, so the
binaries this repository's developers run are never judged.

Ordering is the shared obstacle. A version can be a release tag, a Go pseudo-version, a
`+dirty` build or a Makefile stamp (`<branch>@<git describe --always --dirty>`), and the
places that need an order each handle a subset: the staleness row skips stamps, and the
statusline refresh declines to order a stamp against a release ("cannot order
versions").

G-0147 closed an earlier version of this problem with a Makefile target and a
documented discipline; G-0471 records why that left the hazard unguarded. G-0740
corrected where AI sessions build their diagnostic binary.

## Scope

- One ordering of aiwf version strings, covering release tags, pseudo-versions, `+dirty`
  builds and Makefile stamps, with an explicit "cannot order" outcome, used by every
  place below that compares versions.
- `aiwf update` and `aiwf worktree add` compare an installed artifact with what the
  running binary would write, and do not replace a newer one silently: they refuse or
  report the downgrade, with an explicit override for an intended rollback. Stamping the
  artifact families that carry no version today is part of this work or an explicit
  non-goal decided here (G-0600).
- A mutating verb run inside the aiwf repository by a binary older than the worktree's
  source warns before it writes (G-0471).
- `aiwf doctor`'s staleness row judges the versions it skips today where the ordering can
  order them.
- `make build` stops leaving an executable named like the installed `aiwf` where it is
  reached for instead of the release (G-0654).

## Out of scope

- Upgrading the binary automatically. A stale binary is reported; replacing it stays
  `aiwf upgrade` or `make install`, run by the operator.
- Shipping prebuilt binaries. Distribution is unchanged.
- Bumping the Go toolchain past go1.25.12. The worktree version-stamping G-0740 records
  is fixed by go1.27, a separate decision.
- The planning rituals' reading of a present but stale template (G-0504). Its open
  remainder is a ritual-text requirement that shares none of the version comparison.
- The guidance catalogue's pack set, and the devcontainer kit's toolchains.

## Constraints

- The guards live in the binary. A guarantee that depends on a skill being invoked or a
  convention being followed is not one.
- A refusal aborts before anything is written, so the verb still produces one commit or
  none.
- No guard needs the network. `aiwf update` may already fetch the guidance catalogue,
  but the version comparison never does.
- A pair of versions the ordering cannot order is reported, never treated as a
  downgrade: a false refusal blocks a legitimate update.
- The aiwf-repository-only check (G-0471) is inert in a consumer repository.

## Success criteria

- [ ] An `aiwf update` or `aiwf worktree add` run by a binary older than an installed
  artifact does not replace it silently: the run refuses or reports the downgrade, and an
  explicit override performs an intended rollback.
- [ ] A mutating verb run inside the aiwf repository by a binary older than the
  worktree's source warns before it writes.
- [ ] `aiwf doctor`'s staleness row, the statusline refresh and the downgrade guard
  order the same pair of versions the same way, including Makefile stamps.
- [ ] `make build` no longer leaves an executable named `aiwf` where it is reached for in
  place of the release.
- [ ] Every gap named in *Scope* is addressed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does `update` refuse a downgrade, or report it and proceed, by default? | yes, for the downgrade milestone | A decision record at milestone planning |
| How does a Makefile stamp order against a release or a pseudo-version, given that `git describe` names the nearest tag but not the branch's relation to it? | yes | A decision record before the ordering is built |
| Are the unstamped families (skills, templates, agent cards) stamped, or compared by content? | no | The downgrade milestone's design |
| Does the source comparison run before every mutating verb, or only those that write code-dependent output? | no | Measure its cost at milestone planning |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| An ordering that misreads a newer development build as older blocks a legitimate `update` | high | Unorderable pairs are reported, never refused; the override exists |
| A per-verb source comparison slows every verb in the aiwf repository | med | Measure it; restrict it to the aiwf repository and to mutating verbs |

## Milestones

Candidates, refined by `aiwfx-plan-milestones`:

- Order aiwf version strings, Makefile stamps included · depends on: —
- Stop `update` and `worktree add` silently downgrading installed artifacts · depends on: the ordering
- Warn when a verb in the aiwf repository runs older than the worktree's source, and judge stamps in `doctor` · depends on: the ordering
- Stop `make build` leaving a reachable `bin/aiwf` · depends on: —
