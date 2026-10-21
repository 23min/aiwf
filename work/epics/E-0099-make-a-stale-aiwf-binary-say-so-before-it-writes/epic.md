---
id: E-0099
title: Make a stale aiwf binary say so before it writes
status: active
---

## Goal

An aiwf command that writes the aiwf repository, run by a binary built from other source
than the worktree's, says so before it writes, and `make build` stops leaving such a
binary where it is reached for.

## Context

In the aiwf repository the `aiwf` on `PATH`, or the `bin/aiwf` that `make build`
leaves behind (G-0654), often predates the working tree, and every verb then runs older
logic with no signal (G-0471). `aiwf doctor`'s staleness row
(`internal/cli/doctor/binary_staleness.go`) already compares a binary's commit with the
repository's trunk, but reads the commit only from a Go pseudo-version, so it skips every
`make install` build, whose stamp (`<branch>@<git describe --always --dirty>`) carries
the commit as `g<sha>`.

Every incident G-0600 records happened in this repository: `aiwf update` or
`aiwf worktree add` run by such a binary rewrote materialized artifacts backwards. Each
binary's version could not be ordered against the artifacts' stamp, so a version
comparison in `aiwf update` would have refreshed anyway. Two of those binaries were a
tagged release and a `+dirty` pseudo-version, both of which `binaryStaleness` skips
today.

G-0147 closed an earlier version of this problem with a Makefile target and a
documented discipline; G-0471 records why that left the hazard unguarded. G-0740
corrected where AI sessions build their diagnostic binary.

## Scope

- `aiwf doctor`'s staleness check reads the commit from a Makefile stamp as well as from
  a pseudo-version, and a verb that writes the tree inside the aiwf repository, run by a
  binary whose commit differs from the worktree's, prints that check's warning before
  it writes (G-0471).
- `make build` stops leaving an executable named like the installed `aiwf` where it is
  reached for instead of the release (G-0654).

## Out of scope

- Guarding a consumer repository against `aiwf update` or `aiwf worktree add` run by an
  older release (G-0600). Comparing the guidance fragment's `aiwf-version:` stamp in
  `aiwf update` stops only a downgrade between two tagged releases, which no recorded
  incident shows; it is built when one does.
- A version ordering: a binary is judged by its commit against the worktree's.
- Upgrading the binary automatically, shipping prebuilt binaries, and bumping the Go
  toolchain past go1.25.12.
- The planning rituals' reading of a present but stale template (G-0504), the guidance
  catalogue's pack set, and the devcontainer kit's toolchains.

## Constraints

- Reuse before adding: `binary_staleness.go` for the staleness warning. A new
  comparison needs a case it cannot handle.
- The guards live in the binary. A guarantee that depends on a skill being invoked or a
  convention being followed is not one.
- The warning changes no verb's result: a verb still produces one commit or none, with
  the exit status it has today.
- No guard needs the network.
- The aiwf-repository-only warning (G-0471) is inert in a consumer repository.

## Success criteria

- [ ] A verb that writes the tree inside the aiwf repository, run by a binary whose
  commit differs from the worktree's, warns before it writes, and `aiwf doctor` judges
  `make install` builds as it judges pseudo-versions.
- [ ] `make build` no longer leaves an executable named `aiwf` where it is reached for in
  place of the release.
- [ ] Every gap named in *Scope* is addressed.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does the pre-verb warning run before every mutating verb, or only those whose output depends on the code? | no | Measure its cost when the milestone starts |
| Does the warning also judge a tagged binary by its tag's commit and a dirty build by the commit it was built from? | no | Settle when M-0363 starts; two of G-0600's incidents ran such binaries |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The pre-verb staleness check slows every verb in the aiwf repository | med | It runs only in the aiwf repository and only before mutating verbs; measure it |

## Milestones

- `M-0362` — Stop `aiwf update` silently downgrading installed artifacts · cancelled: it
  guards only a downgrade between two tagged releases, which no recorded incident shows
- `M-0363` — Warn when an aiwf-repository verb runs older than the worktree's source, and stop `make build` leaving a reachable `bin/aiwf` · depends on: —
