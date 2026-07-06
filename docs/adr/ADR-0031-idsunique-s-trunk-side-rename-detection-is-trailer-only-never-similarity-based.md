---
id: ADR-0031
title: idsUnique's trunk-side rename detection is trailer-only, never similarity-based
status: proposed
---
## Context

`idsUnique` (`internal/check/check.go`) reports a `trunk-collision` error when a
working-tree entity and a trunk-ref entity share an id at different paths,
unless a rename explains the divergence. Today it recognizes only a rename
*the branch itself* committed relative to trunk (`gitops.RenamesFromRef`,
scoped `merge-base(HEAD, ref)..HEAD`): a trailer walk
(`aiwf-verb: retitle/rename/reallocate/archive/move`) first, falling back to
`git diff -M` content-similarity when no trailer exists.

`RenamesFromRef`'s one-directional scoping misses the reverse case: a rename
landed *on trunk* (e.g. `aiwf retitle`, which renames the entity's file)
after a branch already forked. G-0378 documents this live — three
independent sessions hit a false-positive `trunk-collision` on the same
stale-branch-vs-trunk-retitle scenario in one day, and the only documented
remediation (`aiwf reallocate <id>`) is actively wrong for it: it creates a
genuine duplicate entity instead of resolving anything.

The first fix considered was a direct mirror: add a trunk-side rename
detector using the identical mechanism (trailer walk + `-M` fallback),
scoped `merge-base(HEAD, ref)..ref`. Three independent adversarial reviews
of that proposal, run before any code was written, found it unsafe as
stated:

- The mirrored `-M` fallback on the trunk side diffs a range that can span
  the trunk's entire history since an old branch forked — potentially
  hundreds or thousands of commits for a long-lived branch, versus the
  branch side's typically-small own-divergence range. aiwf entity files
  share near-identical frontmatter/template boilerplate; over a large
  enough range, `-M`'s similarity heuristic can spuriously pair two
  content-similar but genuinely unrelated entities that were both
  independently touched on trunk during that window.
- A spurious pairing here does not just fail to detect a rename — it
  causes a **real, independent id collision to be silently classified as
  "same entity, moved"** and merged to trunk unnoticed. That is a false
  negative in a correctness gate, and it is strictly worse than the false
  positive being fixed: a false positive blocks a push (loud, recoverable);
  this false negative corrupts the id space silently and permanently.
- Separately, "lazy per disputed id" was shown to invert the cost model on
  the workload the mechanism exists for: a batch rename of ~100+ entities
  on an unmerged branch (`aiwf rewidth --apply`, `aiwf archive --apply`)
  produces one dispute per renamed entity, and evaluating each dispute with
  its own subprocess calls is 50-100x slower than today's single batched
  diff over the same range.
- Subsuming the existing `entity.ActiveFormOf` archive-sweep path
  normalization into the general mechanism was also shown to regress a
  cherry-pick-then-archive edge case it currently handles correctly, for no
  offsetting benefit.

## Decision

`idsUnique`'s rename-based collision exemption is symmetric in *effect*
(either side, branch or trunk, can be recognized as "same entity, moved")
but deliberately asymmetric in *mechanism* between the two sides:

- **Branch side** (unchanged): trailer walk first, `git diff -M`
  content-similarity fallback when no trailer exists. Safe because the
  diffed range is bounded by the branch's own divergence from trunk,
  typically small.
- **Trunk side** (new): trailer walk only. No content-similarity fallback,
  ever, regardless of range size. A trailer is exact, authored-intent
  ground truth (the kernel verb that performed the rename stamped it); a
  similarity diff is a guess. Restricting the trunk side to the exact
  signal makes the false-negative failure mode above structurally
  impossible, at the cost that a manual, non-kernel-verb `git mv` performed
  directly on trunk (bypassing `aiwf retitle`/`aiwf rename`) is not
  auto-recognized and still surfaces as a (safe, recoverable) false-positive
  collision. That cost is consistent with, not introduced by, this repo's
  existing `id-rename-untrailered` check, which already discourages
  non-kernel-verb renames generally.

Supporting decisions, all constraining how G-0378 gets implemented:

- `entity.ActiveFormOf`'s archive-sweep path normalization stays a separate,
  cheap, content-blind, git-free pre-filter, run before any merge-base or
  rename analysis. It is not subsumed into the general mechanism.
- The disputed-id set (same id, different path, between the working tree
  and `Tree.TrunkIDs`) is computed in memory first, at zero git-subprocess
  cost. Empty (the common case, every push) skips all git work entirely.
  Non-empty triggers exactly one batched computation covering every
  disputed id at once — never a per-id loop.
- When `merge-base` itself is unreachable (a shallow clone, or unrelated
  histories), the design defaults to firing the finding — fail-closed on
  uncertainty, an explicit degraded tier rather than a silently-assumed-away
  edge case.
- This design does not claim to eliminate every false positive "by
  construction." A stale branch spanning a trunk-side `aiwf reallocate` (an
  id renumbered on trunk while a branch that referenced the old number is
  unmerged) remains a known, out-of-scope residual limitation.

This work stays deliberately separate from `E-0060` (extending
`refs-resolve`/`body-prose-id` to resolve references to ids absent from the
local tree, via `Tree.LocalRefIDs`/`RemoteRefIDs`). Independent adversarial
review confirmed the two ask genuinely different questions — existence of
an absent id across all refs (E-0060) versus identity of a present-on-both-
sides id (this decision) — at different commit-ishes, with deliberately
different severity semantics per `ADR-0025`/`ADR-0030`'s own axis
separation. One small, non-blocking implementation note carries across
both: reuse the existing (currently-unexported) `trunk.idsFromPaths` helper
for any id-at-a-commit lookup this decision's implementation needs, rather
than re-deriving it, so a later E-0060 milestone doesn't fork a second
version of the same derivation.

## Consequences

- `idsUnique` gains a real fix for the confirmed live bug (a trunk-side
  `retitle`/`rename` no longer false-positives against a stale, unmerged
  branch), without opening the false-negative risk a naive symmetric mirror
  would have.
- A manual, non-kernel-verb rename landed directly on trunk remains an
  unrecognized false positive. Operators hitting it should merge/rebase
  trunk into their branch, or re-perform the rename through the kernel verb
  — not reallocate. (The misleading `aiwf reallocate` hint text for this
  finding is tracked separately as its own gap, since it applies regardless
  of this design's implementation timeline.)
- Implementation needs real git-fixture tests for the merge-base/trailer
  logic (the existing in-memory fake-tree unit tests for the branch-side
  and archive-sweep cases are unaffected and stay as-is).
- The stale-branch-across-a-trunk-reallocate case stays open; a future gap
  can pick it up if it turns out to matter in practice.

## Validation

A fixture test where trunk renames an entity (via a real `aiwf retitle`-
shaped commit) after a branch forks, and the branch never merges it back:
`aiwf check` on the branch must not fire `trunk-collision`. A companion
fixture where two independent entities on trunk and branch coincidentally
share an id, with no rename relationship at all, must still fire — even
when their bodies are highly similar in shape. Both land as acceptance
criteria on whichever milestone implements G-0378.

## References

- G-0378 — the gap this decision resolves the design question for
- ADR-0025 — Allocator's cross-branch view spans all refs, fed to allocation only
- ADR-0030 — Extend cross-branch view to reference resolution and reads
- E-0060 — Resolve cross-branch entity references at check and read time
