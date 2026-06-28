---
id: G-0294
title: Consolidate the three ack-walker HEAD-walk loops into one primitive
status: open
discovered_in: M-0181
---
## Problem

Three ack-walkers in `internal/check` open-code the same HEAD-walk loop:
`WalkAcknowledgedSHAs` and `WalkAcknowledgedSHAEntities` (acks.go) and
`WalkAcknowledgedMistags` (area_mistag.go). Each runs the same
`git log --pretty=format:%H ... %(trailers:unfold=true)` subprocess, splits on
NUL, and loops `gitops.ParseTrailers`; only the per-trailer accumulation
differs. M-0181 added the third clone, crossing the rule-of-three.

## Proposed change

Extract a `forEachCommitTrailers(ctx, root, fn)` primitive (one HEAD-walk +
parse) and reduce all three walkers to short accumulators over it. Pure
refactor; behaviour unchanged, pinned by the existing walker tests.

## Notes

Surfaced by the M-0181 pre-wrap `wf-rethink` of the acknowledge-namespace.
Not blocking: the duplication is ~12 lines of transport framing per walker; the
semantic parsing already routes through the `gitops.ParseTrailers` SSOT. The
single-compute `PolicyAcksHelperLift` does not apply (the mistag set has a
single consumer), so this is a DRY cleanup, not a correctness fix.
