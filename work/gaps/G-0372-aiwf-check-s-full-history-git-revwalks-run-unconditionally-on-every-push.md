---
id: G-0372
title: aiwf check's full-history git revwalks run unconditionally on every push
status: open
---
## What's missing

`aiwf check`'s git-history-dependent rules (`fsm-history-consistent` via
`gitops.BulkRevwalk`, `check.WalkHeadCommits`,
`area_mistag.WalkAcknowledgedMistags`, `orphan_dag`) each walk the entire
reachable commit history from scratch on every invocation, with no notion
of "already verified as of the last check." Measured on this repo (719
entities, 6,213 commits, 43 refs): `aiwf check` costs ~22s unconditionally
on every push, dominated by `BulkRevwalk` (12.2s, of which 3.8s is a `-m`
per-parent merge-diff fan-out that all three known consumers discard
unconditionally via `if o.IsMergeCommit { continue }` at
`internal/check/fsm_history_consistent.go:361,476,689`) and a fully
redundant independent HEAD walk in `WalkAcknowledgedMistags` (1.7s,
`internal/check/area_mistag.go:159`) that duplicates data
`check.WalkHeadCommits` already computed moments earlier in the same
invocation.

A throwaway, read-only prototype (bash, run against this repo's own
history, deleted after use) validated a fix: because git commits are
immutable and content-addressed, a commit's derived observations are
cacheable forever once computed — an exact memoization, not a heuristic.
Measured: a full walk costs 9.46s (6,197 commits); an incremental walk
from a 25-commits-back watermark costs 0.28s (107 commits) — a 33x
speedup — with the union of a cached baseline and the incremental walk
formally verified byte-identical to a fresh full walk (no commit missed,
none double-counted). Full design recorded in
`docs/initiatives/check-performance-incremental-revwalk-cache.md`.

The two independently-fixable findings above have since shipped as a
`wf-patch` (dropping the dead `-m` fan-out; threading the already-computed
`head` slice through `WalkAcknowledgedMistags` instead of re-walking),
cutting real measured wall-clock on this repo from ~25.5s to ~19.1s
(~25% faster, byte-identical findings before/after). This gap stays open
because the structural cause remains: every history-walking rule still
walks the entire reachable history from scratch, on every invocation.

The structural fix was scoped as epic E-0058 and subjected to a four-way
adversarial review, which found it too complex and risky to build as
specified; E-0058 was cancelled. A simpler ref-tip-watermark alternative
(no per-commit cache) was then designed and independently reviewed three
ways; it too was found to have a disqualifying correctness defect —
`fsm-history-consistent`'s verdict depends on the HEAD-relative
acknowledgment set, not just commit content, which any future incremental
design must account for — and was set aside as specified. Both attempts,
the concrete counterexample, and the correctness constraint they surfaced
are recorded in the initiative doc for whoever revisits this next.

## Why it matters

This is the concrete blocker on the branch/id-allocation strategy
discussion in `docs/initiatives/id-lifecycle.md` (the EMB — "ephemeral
mutation branch" — proposal, and G-0281's gaps-inbox side channel): both
assume pushing is cheap enough to retry on contention, which isn't fully
true here yet — every check still pays a cost that scales with total
history, just a smaller one than before the shipped fixes.

The two independently-fixable findings shipped with no design change,
cutting the real cost by about a quarter. The structural fix underneath —
making `aiwf check`'s cost scale with what changed since the last check,
not with total repository history, which only grows over time — remains
unsolved: two attempted designs were set aside (see
`docs/initiatives/check-performance-incremental-revwalk-cache.md`), and
reopening this needs either a design that satisfies the acknowledgment-
interaction constraint that defeated both, or a different approach
entirely.
