---
id: D-0076
title: Measure survivor density over the link primitive
status: proposed
relates_to:
    - G-0630
---
> **Date:** 2026-08-24 · **Decided by:** human/peter

## Question

M-0316/AC-1 sets a surviving-mutant target of at or below 7.7 per thousand lines
across four files — `linkregion.go`, `linkrewrite.go`, `pathrewrite.go` and
`archive.go`. The four came from E-0088's Context, which named them by measured
density before anyone had looked at which mutants the density was made of.

Having now looked: what should the target be measured over, and what counts as a
survivor?

## Decision

Measure over the link primitive — `linkregion.go`, `linkrewrite.go` and
`pathrewrite.go`, 509 lines — and count only survivors carrying no equivalence
argument. `archive.go` leaves the denominator; its survivors are owned by G-0630.

## Reasoning

Measured 2026-08-24 at `419a0890a` with the invocation AC-1 names
(`gremlins unleash --workers 1 --timeout-coefficient 15`, scoped to the four
files): 117 killed, 38 lived, 3 not covered. The 38 survivors split exactly in
half — 19 in the link primitive, 19 in `archive.go`, and every one of
`archive.go`'s is in the archive verb's commit-message builder, planning path or
git-status plumbing. Its link-rewriting functions have zero survivors.

Two things follow, and the first is arithmetic rather than judgment.

`archive.go` is 1,011 of the 1,520 lines in the denominator. Killing every
survivor in the link primitive therefore leaves 19 over 1,520 lines — 12.5 per
thousand, still above the target. **AC-1 as written cannot be satisfied by doing
what the milestone is named for.** It can be satisfied only by additionally
constraining commit-message pluralization and sweep planning, which is a
different subject with a different owner.

Second, 7.7 per thousand lines is a kernel-wide average across packages, and it
has poor resolution applied to individual small files. `linkregion.go` is 142
lines, so the target permits at most one survivor in the entire file; two
mutants argued as equivalent put it at 14.1 and reading as a failure with nothing
wrong. A bar that a correct file cannot clear is measuring the wrong thing.

Counting unexplained survivors rather than raw ones follows the epic's own
constraint, which already admits equivalence with a written argument. Counting
raw would make an argued equivalent indistinguishable from an unexamined one,
which is precisely the distinction AC-2 exists to draw.

Alternatives considered and rejected:

- **Meet AC-1 as written.** Reaches the number, but the milestone's diff becomes
  majority tests for commit-message text — work unrelated to its title, its Goal,
  and the signal that motivated the epic. Hitting a number by building work the
  milestone was not for is the failure mode M-0317 paid for in this same epic.
- **Drop the numeric bar entirely** and keep only AC-2's accounting. Honest, but
  gives up more than the defect requires: the number is recoverable once the
  denominator names the right code, and a falsifiable bar is what the epic asked
  for.
- **Keep four files, exclude equivalents.** Does not help. The 19 `archive.go`
  survivors are unexamined, not argued, so they count either way.

## Consequences

- AC-1's criterion text is restated to match. The before-and-after numbers it
  requires are still recorded with the command that produced them; only the
  denominator and the counting rule change.
- The equivalence arguments become load-bearing. A survivor called equivalent
  without an argument that holds raises the count, so the bar cannot be met by
  assertion.
- G-0630 owns `archive.go`'s 19 survivors, including whether all 19 are worth
  killing. None has been checked for equivalence.
- E-0088's third success criterion reads "across the files named in *Context*",
  which is now wider than AC-1's denominator. That criterion is satisfied by this
  milestone together with G-0630, not by this milestone alone — worth settling at
  epic close rather than leaving the two phrasings to be reconciled by a reader.
