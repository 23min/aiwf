---
id: G-0418
title: Cross-branch hit/collision scan is duplicated across 3 call sites
status: addressed
priority: urgent
discovered_in: M-0260
addressed_by:
    - M-0265
---
## What's missing

The cross-branch scan recipe — "union the local + remote-tracking ref hits,
then run `DetectCollisions` over that same union" — is copied at three
independent call sites: `internal/cli/cliutil/treeload.go` (the `aiwf check`
/ allocation path), `internal/cli/show/show.go`'s `buildCrossBranchShowView`,
and `internal/cli/list/list.go`'s `crossBranchListRows`. The primitives
(`trunk.LocalRefHits`, `RemoteRefHits`, `DetectCollisions`, `DistinctRefs`)
each live once in `internal/trunk`; only the composition — "the hits passed
to `DetectCollisions` must be exactly the union that was scanned" — is
triplicated. All three sites run that scan eagerly over the full union.

## Why it matters — performance

`DetectCollisions` compares blob content for every id that appears on two or
more refs: one `git cat-file` round-trip per hit, each resolving
`<commit>:<path>` (a full tree walk). Over the whole union that is
O(entities × refs). On this repository — 860 entities, 10 refs, ~8300 hits —
a filtered `aiwf list` costs ~24s and `aiwf check` ~57s, and the scan returns
zero cross-branch rows because every id is already present locally (818
distinct ids across all refs equals 818 in the local tree).

The waste is structural: a collision result is consulted only on a local-tree
*miss* (the `refs-resolve` and `body_prose_id` cross-branch branches,
`crossBranchListRows`, and `buildCrossBranchShowView` all guard on a miss
first). Collision-stats spent on locally-present ids — nearly all of them —
are computed and discarded.

## The fix

One trunk-level helper that runs `DetectCollisions` only for ids absent from
the local working tree, consumed by all three sites. This makes the
union/collision coupling atomic (the original duplication concern) and the
scan lazy: collision-stats scale with the locally-absent set, not
entities × refs. In the common all-merged state that is ~zero collision work,
and the filtered list and the check cross-branch scan drop to the ls-tree
floor (~0.3s). Behavior is preserved — the set of cross-branch rows and
findings is unchanged, safe because every consumer is miss-guarded.

Scoped under E-0067's first milestone; the same helper is the seam that makes
G-0416 (collision edit-vs-genuine disambiguation) a cheap successor.
