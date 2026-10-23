# Epic wrap — E-0099

**Date:** 2026-10-23
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0099-make-a-stale-aiwf-binary-say-so-before-it-writes

## Milestones delivered

- M-0363 — Warn when an aiwf-repository verb runs older than the worktree's source
  (merged 33989ef35)

M-0362 — Stop aiwf update silently downgrading installed artifacts — was cancelled; see
*Summary*.

## Changelog entry

### Changed — E-0099: a stale aiwf binary in the aiwf repository says so before every verb

- Inside the aiwf repository, every `aiwf` verb run by a binary whose source differs
  from the worktree's `HEAD` prints `aiwf: warning: stale binary — built from
  <revision>, whose source differs from HEAD` to stderr before it runs. Its output and
  exit status are unchanged, a commit that touches only planning files or `_test.go`
  files does not trigger it, and a consumer repository sees nothing. A plain `go build`
  or `go install` binary run in a linked worktree is not judged, because the go1.25
  toolchain stamps it with the main checkout's commit; build with `make diag-aiwf`
  there.
- `aiwf doctor`'s `binary:` row judges `make install` builds, tagged releases and dirty
  builds by their source against the trunk instead of skipping them.

## Summary

E-0099 set out to make an aiwf binary older than what it acts on say so before it
writes. What shipped is the aiwf-repository half: a binary built from other source
than the worktree's `HEAD` warns before every verb, closing G-0471. M-0362's downgrade
guard for `aiwf update` was cancelled because it stopped only a downgrade between two
tagged releases, which none of G-0600's recorded incidents involved. G-0600 stays open:
`aiwf update` still writes older artifacts over newer ones, after the warning inside the
aiwf repository and with no warning in a consumer repository. Renaming the `bin/aiwf`
that `make build` leaves (G-0654) was reverted because `aiwf doctor --self-check` needs
the binary named `aiwf`.

## ADRs ratified

- none

## Decisions captured

- none — the reasons for cancelling M-0362 and for the scope cuts are in E-0099's
  *Out of scope*, M-0362's cancel commits, and M-0363's design notes, decisions made
  during implementation, and reviewer notes.

## Follow-ups carried forward

- G-0600 — `aiwf update` silently downgrades materialized artifacts when the binary is
  stale; inside the aiwf repository it warns first, in a consumer repository
  nothing warns.
- G-0654 — `make build` emits a `bin/aiwf` that gets reached for instead of the release.

## Doc findings

Clean. The epic's change-set touches two hand-written docs outside `work/`
(`ROADMAP.md` is regenerated): `.guidance/project/environment.md`, where "silently" is
dropped and the closing sentence becomes a pointer at `StaleBinaryWarning` in
`internal/cli/doctor/binary_staleness.go`, which exists; and `TODO.md`, one entry
removed. Neither changes headings or links. `markdownlint` is not installed locally;
CI's markdown-lint job checks both on the push to `main`.

## Handoff

Ready: the pre-verb warning in the aiwf repository and doctor's source-based staleness
row. Left open on purpose: downgrade protection in `aiwf update` (G-0600), the
`bin/aiwf` name (G-0654), and the linked-worktree skip, which goes when the toolchain
stamps worktree builds with their own commit, as go1.27.1 does. A binary built
before this epic — the `aiwf` on `PATH` and the main checkout's `bin/aiwf` — predates
the warning and stays silent until it is rebuilt or deleted.
