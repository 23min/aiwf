---
id: E-0052
title: Broaden the id allocator's cross-branch view to cut collisions
status: done
---
# Broaden the id allocator's cross-branch view to cut collisions

## Goal

Reduce the id-collision *window* mechanically by widening the trunk-aware
allocator from `{working-tree ids + one trunk ref}` to `{working-tree ids + all
local refs + best-effort-fetched trunk}`, so the dominant collision classes are
caught at allocation time instead of surfacing at push via `aiwf reallocate`.
The stable-id-from-creation model is preserved entirely — no inbox, no mint, no
slug phase. `aiwf reallocate` stays the backstop for the irreducible
cross-machine concurrent race.

## Context

`aiwf add` allocates `max(working-tree ids + trunk-ref ids) + 1`, where the trunk
ref defaults to `refs/remotes/origin/main` (`internal/entity/allocate.go`,
`internal/trunk/`). That single trunk ref is the allocator's entire cross-branch
view; it deliberately ignores every other local branch and sibling worktree.
G-0037 (addressed) shipped that trunk-aware allocator and explicitly deferred "an
all-refs walk" as more than that gap required.

The omission bites in practice. G-0272 catalogues three collision classes:

1. **Same machine, multiple worktrees, neither pushed** — the id is sitting in
   the shared local `refs/heads/*`, but the allocator refuses to look. The
   dominant solo+agents case, and the only one that is *artificially* invisible.
2. **Same machine, sequential, trunk-tracking ref stale** — narrowed by a fetch.
3. **Different machines, genuinely concurrent** — unknowable locally; this is
   what `aiwf reallocate` exists for (batch-cure is G-0274's domain).

This epic addresses classes 1 and 2 — the cheap, high-leverage, locally-knowable
ones — without changing the id model. It surfaced as routine friction during
solo+agent work: a worktree-isolated agent session filing entities while the main
checkout does too.

## Scope

### In scope

- **Sibling-worktree local-refs scan (G-0272).** Union ids from all local
  `refs/heads/*` into the allocator's view (and the `ids-unique` trunk-collision
  check). Offline, read-only, cheap — the data is already on disk. Class 1.
- **Opt-in best-effort fetch (G-0273).** Refresh the trunk-tracking ref
  immediately before allocation; best-effort, degrade-to-local on failure, never
  block the add. Class 2.

### Out of scope

- **ADR-0001 (mint entity ids at trunk integration).** The *structural*
  elimination of the collision class via a slug-pre-mint + mint-at-integration
  model. This epic is the cheap, model-preserving point on the same axis;
  ADR-0001 is the heavy structural endpoint. Ratify ADR-0001 at team /
  sustained-parallel-agent scale; until then this epic's broader scan suffices.
  Cheap-now / structural-later on one axis, not competitors.
- **Resolution-side gaps G-0274 (batch reallocate) and G-0308
  (promote-on-wrong-branch mis-attribution after reallocate).** The *cure* side
  of the residual class-3 race; candidate follow-on milestones, held out to keep
  this epic focused on *prevention*.

## Constraints

- Preserve the stable-id-from-creation model: no inbox, no mint, no slug phase,
  no new id shape. The id is allocated and stamped at commit time, as today.
- The local-refs scan is read-only and must degrade cleanly on odd repo states
  (bare, detached HEAD, no branches) — fall back to current behavior, never error.
- The fetch is opt-in and best-effort: a failure (offline, no remote) degrades to
  local-only allocation, never blocks or fails the add.
- `aiwf reallocate` remains the backstop for the irreducible cross-machine
  concurrent race; this epic narrows the window, it does not claim to close it.

## Success criteria

- [ ] The allocator unions all local `refs/heads/*` by default; an id present
      only on a sibling local branch raises the allocated max (no re-allocation).
- [ ] A two-branch integration test demonstrates no collision: one branch
      allocates an id and commits; the next allocation skips past it.
- [ ] An opt-in fetch refreshes the trunk view before allocation, best-effort,
      degrading to local-only offline.
- [ ] The id model is unchanged — no inbox, mint, or slug machinery introduced.

## Milestones

<!-- execution order -->

1. Sibling-worktree local-refs scan (G-0272) — the class-1 cheap big win.
2. Opt-in best-effort fetch-before-allocate (G-0273) — class-2.
