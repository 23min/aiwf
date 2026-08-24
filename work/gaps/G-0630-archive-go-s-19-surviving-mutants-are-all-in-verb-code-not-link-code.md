---
id: G-0630
title: archive.go's 19 surviving mutants are all in verb code, not link code
status: open
discovered_in: M-0316
---
## What's missing

`internal/verb/archive.go` carries 19 surviving mutants, and not one of them is
in the link-rewriting code the file is named for in E-0088's Context. They sit in
the archive verb's own commit-message builder, planning path, skip reporting and
git-status plumbing — code whose tests reach it but do not constrain it.

Measured 2026-08-24 at `419a0890a`, gremlins 0.6.0 under
`gremlins unleash --workers 1 --timeout-coefficient 15`, scoped to the four
E-0088 files by excluding the other 35 production files in `internal/verb`.
Across those four: 117 killed, 38 lived, 3 not covered, efficacy 75.48%.
`archive.go` alone: 55 killed, 19 lived.

The 19 by owning function:

| Survivors | Function |
|---|---|
| 7 | `archiveCommitBody` |
| 4 | `planArchive` |
| 3 | `Archive` |
| 1 each | `computeArchiveMoves`, `archiveCommitSubject`, `declineUndecidableMoves`, `maskedTerminalSkips`, `dirtyEntityPaths` |

The link-primitive functions in the same file — `planArchiveRewrites`,
`linksIntoMove`, `archiveEntityMoves`, `entityBody`, `workingBodyAt` — have zero
survivors between them.

## Why it matters

E-0088 named `archive.go` because its survivor density read as an outlier at
19.1 per thousand lines, and the epic's premise is that three independent signals
agreed on one subsystem. For this file the mutation signal turns out to point at
different code than the other two did: the density comes entirely from the verb
half. The link half is not thereby shown to be well tested — `planArchiveRewrites`,
`linksIntoMove` and `workingBodyAt` carry five mutants between them and all five
die, while `archiveEntityMoves` and `entityBody` carry none at all, so for those
two the run measured nothing either way.

That matters beyond bookkeeping. `archiveCommitBody` composes the commit message
an operator reads to understand what a sweep did, and `aiwf history` renders it
afterwards. Seven unconstrained mutants there means the tests do not pin what
that message says. `planArchive` and `Archive` decide which entities move at all.
A mutant surviving in either is a decision path no assertion holds to its
contract.

Nothing here is a behaviour defect: every one of these mutants survived a green
suite, so the code does what its tests ask. What is missing is that the tests ask
too little.

## Resolution shape

The work is per-function and independent, so it splits rather than needing one
pass. `archiveCommitBody` and `archiveCommitSubject` (8 survivors) are pure text
composition over a move list and are the cheapest to constrain — a table over
move-set shapes asserting the rendered message. `planArchive`, `Archive` and
`computeArchiveMoves` (8) need a tree fixture and assert which entities a sweep
selects. The remaining three sit in skip and git-status paths that already have
fixtures nearby.

Whether all 19 are worth killing is the open question, not a settled target. The
kernel-wide baseline of 7.7 per thousand lines is an average across packages, and
applying it per file has poor resolution at these sizes — a decision recorded
under M-0316 declines to use it as this file's bar for that reason. Some of the
19 may also be equivalent mutants; none has been checked, because M-0316 scoped
itself to the link primitive.

## Where to fix

- `internal/verb/archive.go` — the functions listed above.
- `internal/verb/archive_test.go` and the archive-specific test files beside it —
  where the constraining assertions would land.
- `.github/workflows/mutate-hunt.yml` — the invocation that reproduces the
  measurement; its header explains why `--workers 1` and
  `--timeout-coefficient 15` are set as they are.
