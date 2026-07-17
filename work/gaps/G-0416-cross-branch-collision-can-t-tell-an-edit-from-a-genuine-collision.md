---
id: G-0416
title: Cross-branch-collision can't tell an edit from a genuine collision
status: open
priority: low
discovered_in: M-0259
---
## Problem

`trunk.DetectCollisions` (M-0259/AC-3) classifies any id whose
cross-branch hits carry divergent blob content as `cross-branch-collision`.
Per D-0036, this finding is deliberately non-blocking (`SeverityWarning`)
because blob-SHA divergence alone cannot distinguish two different
situations that produce identical evidence:

1. A genuine duplicate-mint collision — two refs independently
   allocated different entities under the same id.
2. An ordinary same-entity edit landed on one of several refs sharing
   the id's history, still unmerged (the common case given aiwf's own
   recommended multi-worktree workflow, since linked worktrees share
   local branch refs).

D-0036 accepts this coarse v1 semantics rather than trying to
disambiguate, since the genuine case (1) is still caught — just later,
by the pre-existing `ids-unique`/`trunk-collision` check once both
copies land in a shared tree — and no cheap heuristic reliably
distinguishes the two (a "does it match trunk" heuristic breaks down as
soon as two branches independently edit the same entity for unrelated
reasons: neither matches trunk, neither matches the other, and it's
genuinely ambiguous whether that's two edits or a collision).

## Direction

The only fully correct fix is git-lineage disambiguation: for a
diverging pair of hits, determine via `git merge-base` (already a
primitive this codebase uses — the `aiwf reallocate` tie-breaker)
whether the two refs' copies of the entity share a common ancestor
blob. If they do, it's the same entity, edited (case 2, stays
cross-branch-pending); if they don't, it's a genuine duplicate mint
(case 1, warrants a stronger signal than the current warning).

This is real new scope, not a quick patch: a second git primitive
integration, new edge cases (renames landing between the merge-base and
either tip, three or more diverging refs, a merge-base that doesn't
resolve), and its own TDD cycle. Proportionate as a follow-on
milestone under E-0060 or a successor epic, should the coarse v1
severity prove insufficient in practice (e.g. genuine collisions going
unnoticed because operators stop reading warnings).

## Provenance

Surfaced by an independent code-quality review of M-0259 at wrap
(2026-07-16), resolved into D-0036's accepted decision in the same
session.
