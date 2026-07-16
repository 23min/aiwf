---
id: M-0259
title: Add cross-branch-pending tier and collision detection to reference checks
status: in_progress
parent: E-0060
tdd: required
acs:
    - id: AC-1
      title: Cross-branch view carries per-id path and ref
      status: met
      tdd_phase: done
    - id: AC-2
      title: Local-tree miss resolves as cross-branch-pending, not unresolved
      status: met
      tdd_phase: done
    - id: AC-3
      title: Divergent content across refs escalates to cross-branch-collision
      status: open
      tdd_phase: done
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

Classify a reference (structured field or prose token) to an id that exists
only on another local branch or remote-tracking ref as a distinct,
non-blocking `cross-branch-pending` finding instead of a hard `unresolved`
— and correctly tell that case apart from a genuine cross-branch collision.

## Context

ADR-0030 records the decision this milestone implements. `E-0052` (done)
already built the cross-branch view this milestone widens and consumes
(`Tree.LocalRefIDs`/`RemoteRefIDs`, M-0212/M-0214); `G-0241` shipped the
precedent second-tier resolver shape (the silent trunk fallback in
`classifyBodyToken`). `G-0415`, filed while analyzing this epic, adds the
collision-divergence requirement (AC-3) and the accepted-limitation note
below — this milestone addresses it.

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

## Constraints

- Reuse `Tree.LocalRefIDs`/`Tree.RemoteRefIDs`/`AllocationIDs()` as-is where
  possible; AC-1's widening is additive and must not change the allocator's
  existing consumption of them.
- No entity content is copied, cached, or materialized into the working
  tree, the index, or a new ref — AC-3's blob-SHA comparison reads live via
  `gitops.BlobReader`, never `git checkout`/`git merge`.
- Transient git scan failures in the underlying `LocalRefIDs`/`RemoteRefIDs`
  collection (an individual ref that lists but fails to read mid-scan) are
  accepted as a documented, self-healing limitation for v1: no retry, no
  new error-signaling plumbing added to the shared allocator-facing
  primitives. A spurious `unresolved` from a one-off race clears on the
  next `aiwf check` run, since nothing here is cached (`G-0415`).

## Design notes

- The new `cross-branch-pending`/`cross-branch-collision` classifications
  land as subcodes on the existing `refs-resolve`/`body-prose-id` finding
  codes (ADR-0030's Decision section), not new finding codes — mirrors how
  the existing `Trunk` tier reuses the same codes.
- Collision detection (AC-3) is a blob-SHA comparison via the existing
  `gitops.BlobReader`, not a content diff or merge simulation — cheap and
  precise: identical SHA means identical content, no ambiguity possible.

## Surfaces touched

- `internal/trunk/trunk.go` (`LocalRefIDs`/`RemoteRefIDs`/`refIDs`)
- `internal/check/check.go` (`refsResolve`)
- `internal/check/body_prose_id.go` (`classifyBodyToken`, `BodyProseIDIndex`)
- `internal/gitops/catfile.go` (`BlobReader`, consumed not modified)

## Out of scope

- Read-side rendering (`aiwf show`/`aiwf list`) — `M-0260`.
- Any mutating verb accepting a `cross-branch-pending` or
  `cross-branch-collision` target (epic-level out of scope, unchanged).
- Changes to `ids-unique`'s trunk-anchored basis or `TrunkIDs`'s existing
  silent resolution tier (`G-0241`) — unchanged.

## Dependencies

- `E-0052` / `M-0212` / `M-0214` (done) — the cross-branch view this
  milestone widens.
- `G-0241` (addressed) — precedent resolver shape.
- `ADR-0030` (proposed) — implements its decision; should be accepted
  before or alongside this milestone landing.
- `G-0415` (open) — addressed by this milestone landing.

## References

- ADR-0030 — Extend cross-branch view to reference resolution and reads
- ADR-0025 — Allocator's cross-branch view spans all refs, fed to
  allocation only
- E-0052 — Broaden the id allocator's cross-branch view to cut collisions
- G-0241 — BodyProseIDIndex skips TrunkIDs; trunk-only ids appear
  unresolved
- G-0415 — Cross-branch reference resolution must detect same-id
  divergence across refs

---

## Work log

### AC-1 — Cross-branch view carries per-id path and ref

Widened `LocalRefIDs`/`RemoteRefIDs` to `LocalRefHits`/`RemoteRefHits`
returning `[]trunk.RefHit{Kind, ID, Path, Ref}`; the two string-slice
functions are now thin derived wrappers so the allocator's existing
consumption is unaffected · commit 76c1a712 · tests 5/5 new (plus 8
existing M-0212/M-0214 tests passing unmodified)

### AC-2 — Local-tree miss resolves as cross-branch-pending, not unresolved

Added `Tree.CrossBranchHits` (populated once per side in
`LoadTreeWithTrunk`, no duplicate scan) and a shared
`crossBranchIndex`/`joinRefNames` helper; `refsResolve` and
`classifyBodyToken` now consult it on a local-tree miss before firing
`unresolved`, emitting the non-blocking `cross-branch-pending` subcode
instead (`SeverityWarning`). Composite-id resolution (`M-NNN/AC-N`)
deliberately stays out of scope — validating a composite's AC position
would require reading the target's content from another ref, which is
read-side territory (`M-0260`) · commit 138c5e43 · tests 12/12 new

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)

