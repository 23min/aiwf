---
id: M-0316
title: Kill the link primitive's surviving mutants
status: in_progress
parent: E-0088
depends_on:
    - M-0315
tdd: none
acs:
    - id: AC-1
      title: Survivor density in the link subsystem meets the kernel baseline
      status: open
    - id: AC-2
      title: Every remaining survivor is recorded as equivalent or tracked
      status: open
---

## Goal

Bring the link subsystem's tested edges up to the standard the rest of the
kernel already meets, using surviving mutants as the work list and the measure.

## Context

Mutation testing across six packages and roughly 28,000 production lines
established a kernel-wide baseline of 7.7 surviving mutants per thousand lines,
with per-package efficacy between 88.6% and 96.6%. The link subsystem is the
outlier: `linkregion.go` measures 70.4 survivors per thousand lines — the
highest density in the kernel — with `linkrewrite.go` at 30.9, `pathrewrite.go`
at 21.1 and `archive.go` at 19.1.

`archive.go`'s share of that is not link code. Its link-rewriting functions —
`planArchiveRewrites`, `linksIntoMove`, `archiveEntityMoves`, `entityBody`,
`workingBodyAt` — have no survivors between them; the density comes from the
archive verb's commit-message builder, planning path and git-status plumbing,
which G-0630 owns. AC-1 measures the link primitive alone for that reason
(D-0076).

The link primitive's 19 survivors are 11 conditional-boundary, 4 invert-negatives
and 4 arithmetic-base, with none surviving from conditional negation. The
boundary majority is the signature of a happy path under test and edges that are
not, and it matches the two defects E-0088's earlier milestones fix: both were
edge cases that the existing tests walked past.

This milestone runs last of the code milestones so the outbound paths M-0315
adds are measured alongside the rest rather than becoming a fresh untested
surface.

## Acceptance criteria

### AC-1 — Survivor density in the link subsystem meets the kernel baseline

Unexplained-survivor density across the link primitive — `linkregion.go`,
`linkrewrite.go` and `pathrewrite.go`, 509 lines — is at or below 7.7 per
thousand lines, measured by the same gremlins invocation that established the
baseline (`--workers 1 --timeout-coefficient 15`). A survivor counts until it
carries the equivalence argument AC-2 requires, so the bar cannot be met by
assertion. The before and after numbers are both recorded with the command that
produced them.

`archive.go` is outside the denominator per D-0076: every one of its survivors
is in the archive verb rather than the link primitive, and G-0630 owns them.

### AC-2 — Every remaining survivor is recorded as equivalent or tracked

No survivor is left unexplained. Each remaining one is either recorded as an
equivalent mutant with the argument for why the mutation cannot change observable
behavior, or tracked as work with its own entity. A survivor count that falls
without an account of what remains does not satisfy this.

## Constraints

- **Measure, do not assert.** A claim that a survivor is dead names the command
  and its output. This milestone's whole deliverable is a measurement, so an
  unmeasured claim is the failure mode.
- **Equivalence needs an argument.** Naming a mutant equivalent requires saying
  why the mutation cannot change observable behavior, not that a test was hard
  to write.
- **Tests pin behavior, not implementation.** A test written to kill a specific
  mutant must still assert what the code does for given inputs.

## Design notes

Two known measurement traps apply to this milestone and will otherwise mislead
it. Gremlins places mutants inside multi-clause `case` conditions while Go
instruments the case body, so switch-dense code reports phantom "not covered"
mutants — treat that column as an upper bound and cross-check against
`go tool cover`. Separately, gremlins sees only a package's own tests unless
`--coverpkg` is set, so efficacy is trustworthy only where tests are co-located
with the code. Both traps produced false findings during E-0088's planning.

## Out of scope

- Production behavior changes. If killing a mutant requires changing behavior
  rather than adding a test, that is a defect and belongs in its own entity.
- Packages outside the four named files.

## Dependencies

- M-0315 — its outbound paths are part of what this milestone measures.

## References

- E-0088 — the parent epic, which records the baseline and the per-file densities
- D-0076 — why AC-1 measures the link primitive rather than the four named files
- G-0630 — `archive.go`'s survivors, which are archive-verb code
- `internal/verb/linkregion.go`, `linkrewrite.go`, `pathrewrite.go` — the surface
  AC-1 measures; `archive.go` is measured for AC-2's accounting only
- `.github/workflows/mutate-hunt.yml` — the invocation and why its flags are set
