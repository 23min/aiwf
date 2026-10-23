---
id: E-0099
title: Make a stale aiwf binary say so before it writes
status: active
---

## Goal

An aiwf command run in the aiwf repository by a binary built from other source than the
worktree's says so before it runs.

## Context

In the aiwf repository the `aiwf` on `PATH`, or the `bin/aiwf` that `make build`
leaves behind (G-0654), often predates the working tree, so a verb can run older logic
than the source in front of it (G-0471). `aiwf doctor`'s staleness row
(`internal/cli/doctor/binary_staleness.go`) is the check the warning reuses; a
`make install` build's stamp (`<branch>@<git describe --always --dirty>`) carries its
commit as `g<sha>`.

Every incident G-0600 records happened in this repository: `aiwf update` or
`aiwf worktree add` run by such a binary rewrote materialized artifacts backwards. Each
binary's version could not be ordered against the artifacts' stamp, so a version
comparison in `aiwf update` would have refreshed anyway. Two of those binaries were a
tagged release and a `+dirty` pseudo-version, which is why the staleness check judges
those by their commit as well.

G-0147 closed an earlier version of this problem with a Makefile target and a
documented discipline; G-0471 records why that left the hazard unguarded. G-0740
corrected where AI sessions build their diagnostic binary.

## Scope

- `aiwf doctor`'s staleness check reads the revision from every binary that carries
  one — a pseudo-version, a Makefile stamp, a tagged release, a dirty build — and every
  verb run inside the aiwf repository by a binary whose source differs from the
  worktree's `HEAD` prints that check's warning before it runs, except a plain
  `go build` or `go install` binary in a linked worktree (G-0471).

## Out of scope

- Guarding a consumer repository against `aiwf update` or `aiwf worktree add` run by an
  older release (G-0600). Comparing the guidance fragment's `aiwf-version:` stamp in
  `aiwf update` stops only a downgrade between two tagged releases, which no recorded
  incident shows; it is built when one does.
- A version ordering: a binary is judged by whether its source matches the worktree's.
- Renaming the `bin/aiwf` that `make build` leaves (G-0654): `make selfcheck` depends on
  the name, and a copy built from this repository warns once its source falls behind.
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

- [ ] Every verb run inside the aiwf repository by a binary whose source differs from
  the worktree's `HEAD` warns before it runs, a plain `go build` or `go install` binary
  in a linked worktree excepted, and `aiwf doctor` judges `make install`
  builds, tagged releases and dirty builds by the source they were built from.
- [ ] Every gap named in *Scope* is addressed.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The pre-verb staleness check slows every verb in the aiwf repository | low | It runs only in the aiwf repository; M-0363 records its measured cost |

## Milestones

- `M-0362` — Stop `aiwf update` silently downgrading installed artifacts · cancelled: it
  guards only a downgrade between two tagged releases, which no recorded incident shows
- `M-0363` — Warn when an aiwf-repository verb runs older than the worktree's source · depends on: —
