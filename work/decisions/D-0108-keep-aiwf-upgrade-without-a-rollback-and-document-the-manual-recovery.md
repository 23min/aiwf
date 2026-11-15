---
id: D-0108
title: Keep aiwf upgrade without a rollback and document the manual recovery
status: accepted
relates_to:
    - D-0043
    - G-0430
    - G-0600
    - M-0270
---
> **Date:** 2026-11-15 · **Decided by:** human/peter

## Question

`aiwf upgrade` installs a release with one `go install` and keeps no copy of the
binary it replaces, so it has no rollback. D-0043 chose to track that absence as a
gap, G-0430, rather than as prose in `CLAUDE.md` or `aiwf upgrade --help`. Its
Consequences say G-0430 "stays open until either a rollback capability ships
(closing the gap alongside a real `--help` update) or a future decision
explicitly judges minimalism the right permanent call". Is minimalism the right
permanent call?

## Decision

- Build no rollback into `aiwf upgrade`. Minimalism is the permanent call.
- State the recovery in the README's "Upgrade" section: while the installed
  binary still runs, `aiwf upgrade --version vX.Y.Z` installs the earlier release
  and refreshes the consumer repo; when it does not, reinstall the earlier release
  with `go install github.com/23min/aiwf/cmd/aiwf@vX.Y.Z` and run `aiwf update` in
  each consumer repo.
- Close G-0430 as addressed by this decision.

## Reasoning

A recovery already exists. `aiwf upgrade --version` with an older version goes
ahead as a downgrade, installing that release and re-running `aiwf update`; when
the newer binary is too broken to run, `go install` of the earlier tag followed by
`aiwf update` does the same by hand, since `aiwf update` writes the skills and
guidance (the property G-0600 records) and the hooks from whichever binary runs
it. For released versions, a backup-and-restore path inside `aiwf upgrade` would
duplicate that for a failure G-0430 records no instance of.

D-0043 turned down `CLAUDE.md` because it never reaches the operators who run
`aiwf upgrade`, and `--help` because documenting the absence of a `--rollback`
flag "reads as aspirational rather than descriptive, and drifts the moment a real
decision gets made about whether to build one". This is that decision, so a
sentence about recovery now describes the settled state. The README answers both:
an operator reads it when upgrading, and it stays readable when the installed
binary is too broken to print its own help.

G-0430 calls itself "not necessarily a defect" and anticipated this answer; closed
by this decision, it remains the referenceable record it said it would be.

## Consequences

- Both recovery paths rely on `aiwf update` writing an earlier release's artifacts
  over newer ones, which is what G-0600 records as a defect. Its fix must keep a
  deliberate way to do that, such as the override G-0600 already names, and the
  README paragraph names that way when the fix ships.
- `TestM0270_AC6_GapTracksUpgradeRollbackAbsence` pins G-0430 as `open`. It changes
  in the same patch as the README so that it accepts G-0430 either `open` or
  `addressed` with an accepted decision in `addressed_by`, keeping its guard
  against the gap closing without one.
- A user-level statusline installed by a newer release is not downgraded by the
  older binary's `aiwf update`.
