---
id: M-0259
title: Add cross-branch-pending tier and collision detection to reference checks
status: draft
parent: E-0060
tdd: required
acs:
    - id: AC-1
      title: Cross-branch view carries per-id path and ref
      status: open
      tdd_phase: red
    - id: AC-2
      title: Local-tree miss resolves as cross-branch-pending, not unresolved
      status: open
      tdd_phase: red
    - id: AC-3
      title: Divergent content across refs escalates to cross-branch-collision
      status: open
      tdd_phase: red
    - id: AC-4
      title: Escalation re-fires unresolved once the source branch disappears
      status: open
      tdd_phase: red
    - id: AC-5
      title: An id absent everywhere still hard-fails unresolved
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — Cross-branch view carries per-id path and ref

The cross-branch view carries `(kind, id, path, ref)` per hit instead of
collapsing to bare id strings. `internal/trunk/trunk.go`'s `idsFromPaths`
already computes kind and path per hit; `refIDs()` discards everything but
the id string and never records which ref a hit came from. Widen the
consumed shape to keep path and tag the originating ref — additive only:
`AllocationIDs()`'s existing `[]string` consumption (the allocator) is
unaffected, and no new git-scanning mechanism is introduced (the same
`git ls-tree` calls already made).

Evidence: a unit test asserting the widened view carries path and ref for a
fixture id, plus the existing M-0212/M-0214 allocator tests passing
unmodified (proving the widening didn't change allocator-facing behavior).

### AC-2 — Local-tree miss resolves as cross-branch-pending, not unresolved

`refs-resolve` (structured fields) and `body-prose-id` (prose tokens)
consult AC-1's widened cross-branch view on a miss against the local
tree/trunk tiers, before falling through to `unresolved`. A hit — a single
ref, or multiple refs agreeing on content per AC-3 — classifies as a new,
non-blocking subcode `cross-branch-pending` on the existing
`refs-resolve`/`body-prose-id` finding codes, mirroring the shape of the
existing silent `Trunk` tier in `classifyBodyToken`
(`internal/check/body_prose_id.go`) but visible rather than silent, per
ADR-0030.

Evidence: fixture test — an id exists only on a sibling local branch; a
reference to it from the working tree classifies `cross-branch-pending`;
`aiwf check`'s exit status stays non-blocking for this subcode alone.

### AC-3 — Divergent content across refs escalates to cross-branch-collision

When AC-1's view finds an id on more than one ref, compare the blob SHA at
each ref's recorded path via `gitops.BlobReader` (`git cat-file --batch`,
already used by `internal/check/fsm_history_consistent.go` — no new git
primitive). Identical SHA across every ref holding the id stays
`cross-branch-pending` — one entity, just not merged yet, nothing
ambiguous. Divergent SHA escalates to a distinct subcode,
`cross-branch-collision`, instead of being silently classified as the soft
tier. Resolves G-0415's multiplicity gap: today's `ids-unique` check never
compares sibling local branches against each other, so this is the first
surface that can detect a genuine cross-branch collision before either
side merges.

Evidence: fixture test with two local branches independently holding the
same id with different content; the reference resolves
`cross-branch-collision`, not `cross-branch-pending`.

### AC-4 — Escalation re-fires unresolved once the source branch disappears

Per ADR-0030's Validation section: a reference validated
`cross-branch-pending` while its source branch exists must re-escalate to
`unresolved` once that branch disappears from the cross-branch view too
(deleted, abandoned, never merged). This falls out of AC-1's view being
recomputed live from current git refs on every `aiwf check` run — nothing
is cached, so there is no separate escalation-tracking mechanism to drift.

Evidence: the fixture test itself — reference classifies
`cross-branch-pending` while the source branch exists; branch deleted;
`aiwf check` re-run; the same reference now reports `unresolved` — run in
CI on every pass.

### AC-5 — An id absent everywhere still hard-fails unresolved

A reference to an id found in neither the local tree/trunk tiers nor AC-1's
cross-branch view is unchanged from today: it still fails `unresolved`,
exactly as before this milestone. This is the guard against the new tier
ever softening a genuinely fabricated or deleted id — the one case ADR-0030
explicitly keeps hard-failing.

Evidence: existing `unresolved` fixture tests continue to pass unmodified;
a new fixture test with a fabricated id present nowhere (not local tree,
not trunk, not any local/remote ref) confirms the finding subcode stays
`unresolved`.

