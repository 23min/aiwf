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
`PolicyCoherenceGuardChokepoint` re-derives the same claim by its own AST scan,
with the seam's identity re-encoded inline rather than read from the shared
constants.

The genuinely new assertions in the second are the guard-presence field and a
wider reach: it resolves the gitops import's local name, so an aliased import is
caught, and it watches every commit-construction primitive rather than only the
one a verb is meant to call. Both of those are properties the older policy
should have and does not.

## Why it matters

One fact, two implementations, and they can drift apart one plausible edit at a
time — the case H1 exists to prevent. The copies already differ in scope: the older policy
filters to two package prefixes, the newer walks the tree; the older resolves
no import alias and watches a narrower primitive set. Those differences are not
principled, they are artifacts of writing the second one separately — and the
newer one is now the stricter, which means the fold has a direction.

The older policy's narrower scope is itself a known hole, recorded when it was
written: a second caller appearing in another package would not be caught. So
merging is not only deduplication — it closes that hole as a side effect,
because the surviving policy would inherit the wider scan.

Two smaller things belong with the fold rather than on their own. The newer
policy's roster of gitops commit primitives is hand-maintained with no owner and
no retirement trigger — it is complete today, but a fifth exported primitive
would leave its reach silently. Deriving the set from the package's exported
surface is the census the fold could carry. And the scan resolves direct calls
only: a primitive taken as a function value, held in a variable, passed as an
argument, or reached through a dot-import is not seen. None of those appears in
the tree and each is a strange way to build a commit, but a caller intent on
bypassing the seam has them.

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
