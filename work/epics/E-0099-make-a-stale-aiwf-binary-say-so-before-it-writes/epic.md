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
the installed artifacts moves them backwards and reports `updated` (G-0600).

Two pieces of the answer already exist. The guidance fragment carries the version of the
binary that last wrote it (`aiwf-version:`), and every `update` rewrites it with the
other artifacts, so its stamp stands for the run. The statusline refresh already guards
its own file with `version.Compare(binary, installed)` in
`decideStatuslineRefresh` (`internal/skills/statusline.go`): an older binary skips the
refresh and says so.

In the aiwf repository itself the `aiwf` on `PATH`, or the `bin/aiwf` that `make build`
leaves behind (G-0654), often predates the working tree, and every verb then runs older
logic with no signal (G-0471). `aiwf doctor`'s staleness row
(`internal/cli/doctor/binary_staleness.go`) already compares a binary's commit with the
repository's trunk, but reads the commit only from a Go pseudo-version, so it skips every
`make install` build, whose stamp (`<branch>@<git describe --always --dirty>`) carries
the commit as `g<sha>`.

G-0147 closed an earlier version of this problem with a Makefile target and a
documented discipline; G-0471 records why that left the hazard unguarded. G-0740
corrected where AI sessions build their diagnostic binary.

## Scope

- `aiwf update` compares the guidance fragment's stamp with the running binary using
  `version.Compare`, as the statusline refresh does, before it writes. A binary older
  than the stamp skips the artifact refresh and reports it; a pair `version.Compare`
  cannot order proceeds and reports that it could not compare (G-0600).
- `aiwf doctor`'s staleness check reads the commit from a Makefile stamp as well as from
  a pseudo-version, and a mutating verb run inside the aiwf repository by a binary
  older than the worktree's source prints that check's warning before it writes
  (G-0471).
- `make build` stops leaving an executable named like the installed `aiwf` where it is
  reached for instead of the release (G-0654).

## Out of scope

- A version ordering beyond `version.Compare`. Releases order through it; a Makefile
  stamp or pseudo-version is judged by its commit in the aiwf repository and reported
  as unorderable elsewhere.
- Version stamps on skills, templates or agent cards. The fragment's stamp stands for
  the run that wrote them.
- An override that forces a downgrade. Add one when an intended rollback needs it.
- A separate guard in `aiwf worktree add`. A new worktree has no installed ignored
  artifacts to downgrade; the tracked block it writes into `AGENTS.md` shows as a `git
  diff`, and in the aiwf repository the pre-verb warning fires first.
- Upgrading the binary automatically, shipping prebuilt binaries, and bumping the Go
  toolchain past go1.25.12.
- The planning rituals' reading of a present but stale template (G-0504), the guidance
  catalogue's pack set, and the devcontainer kit's toolchains.

## Constraints

- Reuse before adding: `version.Compare` and `decideStatuslineRefresh`'s rule for the
  downgrade check, and `binary_staleness.go` for the staleness warning. A new comparison
  needs a case these cannot handle.
- The guards live in the binary. A guarantee that depends on a skill being invoked or a
  convention being followed is not one.
- A skipped refresh writes nothing for the artifacts it skips, and a verb still produces
  one commit or none.
- No guard needs the network.
- A pair `version.Compare` cannot order is reported, never treated as a downgrade.
- The aiwf-repository-only warning (G-0471) is inert in a consumer repository.

## Success criteria

- [ ] An `aiwf update` run by a release older than the release that last wrote the
  installed artifacts leaves them unchanged and says so.
- [ ] A mutating verb run inside the aiwf repository by a binary older than the
  worktree's source warns before it writes, and `aiwf doctor` judges `make install`
  builds as it judges pseudo-versions.
- [ ] `make build` no longer leaves an executable named `aiwf` where it is reached for in
  place of the release.
- [ ] Every gap named in *Scope* is addressed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the pre-verb warning run before every mutating verb, or only those whose output depends on the code? | no | Measure its cost when the milestone starts |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| A skipped refresh hides a real fix from an operator who meant to roll back | low | The report names the skip and the newer stamp; an override is added when that case arises |
| The pre-verb staleness check slows every verb in the aiwf repository | med | It runs only in the aiwf repository and only before mutating verbs; measure it |

## Milestones

The two are independent and can run in either order.

- `M-0362` — Stop `aiwf update` silently downgrading installed artifacts · depends on: —
- `M-0363` — Warn when an aiwf-repository verb runs older than the worktree's source, and stop `make build` leaving a reachable `bin/aiwf` · depends on: —

