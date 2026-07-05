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
none double-counted). Full design — per-ref watermarks, force-push/
rewrite invalidation via `merge-base --is-ancestor` checks, a
schema-version stamp for logic-change invalidation, and a fail-safe-to-
full-walk posture on any doubt — recorded in
`docs/initiatives/check-performance-incremental-revwalk-cache.md`.

## Why it matters

This is the concrete blocker on the branch/id-allocation strategy
discussion in `docs/initiatives/id-lifecycle.md` (the EMB — "ephemeral
mutation branch" — proposal, and G-0281's gaps-inbox side channel): both
assume pushing is cheap enough to retry on contention, which isn't true
here today — every retry pays the full ~22s tax regardless of what
changed. Until this is fixed, "push often" (the cheapest available
mitigation for main-moved contention) is itself expensive, and any id or
branch mechanism built on top of frequent pushing inherits the same cost
rather than solving it.

Two of the findings are independently fixable today with no design
change, worth ~5.5s (~25% of the total) combined: dropping the dead `-m`
fan-out, and threading the already-computed `head` slice through
`WalkAcknowledgedMistags` instead of re-walking. The incremental-cache
proposal is the structural fix underneath both — it makes `aiwf check`'s
cost scale with what changed since the last check, not with total
repository history, which only grows over time under the current design.
