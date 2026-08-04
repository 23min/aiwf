---
id: G-0545
title: Fold the coherence-guard seam policy into the commit-construction policy
status: open
discovered_in: M-0291
---
## What's missing

Two policies now assert the same property about the same seam.
`PolicyCommitConstructionSingleSeam` already holds that nothing outside
`verb.Apply` calls `gitops.CommitVerbChange`, by AST selector inspection with
the seam's identity in named constants.
`PolicyCoherenceGuardChokepoint` re-derives that claim by substring match over
raw function-body text, with the seam's identity re-encoded inline.

The genuinely new assertion in the second is one field: whether the seam's body
also names the sovereign-force guard.

## Why it matters

One fact, two implementations, and they can drift apart one plausible edit at a
time — the case H1 exists to prevent. The copies already differ in mechanism
and in scope: the older policy filters to two package prefixes, the newer walks
the tree. That difference is not principled, it is an artifact of writing the
second one separately.

The older policy's narrower scope is itself a known hole, recorded when it was
written: a second caller appearing in another package would not be caught. So
merging is not only deduplication — it closes that hole as a side effect,
because the surviving policy would inherit the wider scan.

## Options

1. **Fold the guard-presence clause into `PolicyCommitConstructionSingleSeam`
   and widen its scan to the tree.** One policy, one seam constant, one walk,
   and the pre-existing scope hole closes with it. The cost is touching a
   policy that several ACs already cite.
2. **Leave both and cross-reference them.** Cheapest now, and the drift risk
   stays.

Option 1 is the lean, as its own patch with its own review rather than folded
into a wrap.

## Scope

Surfaced by both lenses of M-0291's wrap review, independently. The duplication
is deliberate to the extent that the newer policy was written without checking
for the older one — which is the H1 failure mode rather than a decision.
