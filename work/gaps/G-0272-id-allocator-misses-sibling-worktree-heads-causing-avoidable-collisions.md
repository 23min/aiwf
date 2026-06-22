---
id: G-0272
title: ID allocator misses sibling worktree heads, causing avoidable collisions
status: open
---
## Problem

`aiwf add` allocates the next id as `max(ids in working tree ∪ ids in one trunk
ref) + 1`, where the trunk ref defaults to `refs/remotes/origin/main`
(`internal/entity/allocate.go`, `internal/trunk/trunk.go`,
`config.DefaultAllocateTrunk`). That single trunk ref is the allocator's entire
cross-branch view — it deliberately ignores every other local branch and every
sibling git worktree.

The consequence: two allocation contexts that both observe the same `max`, where
neither has published, produce the same id. The collision stays invisible until
one side lands on trunk and the other's next push hits the `ids-unique` pre-push
check — then it has to be cured with `aiwf reallocate`.

The dominant real-world instance is entirely self-inflicted. Git worktrees of one
repo share the same object store and `refs/heads/*`. When a solo operator runs
parallel sessions across several worktrees ("same brain, different sessions,
hours apart, neither pushed yet" — CLAUDE.md), the sibling worktree's
freshly-allocated id is sitting right there in the shared local refs. The
allocator just refuses to look at it, because it only reads the trunk ref.

## Why this is the cheap, high-leverage fix

Collisions split into three classes:

1. **Same machine, multiple worktrees, neither pushed** — knowable from local
   refs, but the allocator doesn't look.
2. **Same machine, sequential, trunk-tracking ref stale** — narrowed by a fetch
   (`G-0273`).
3. **Different machines, genuinely concurrent** — unknowable locally; this is
   what `aiwf reallocate` exists for (batch-cure tracked in `G-0274`).

Class 1 is the one CLAUDE.md flags as most frequent here, and it is the only one
that is *artificially* invisible: the data is already on disk. The recorded
rationale ("does not scan parallel feature branches") is really about class 3 and
over-generalizes onto class 1.

## Direction (to converge at the milestone)

- Union the allocation scan with the HEADs of all live worktrees
  (`git worktree list`; machinery exists in `internal/gitops/worktrees.go`),
  not the trunk ref alone.
- Prefer scoping to *worktree* HEADs over all `refs/heads/*`: a long-dead branch
  should not permanently reserve an id and inflate the counter, leaving gaps in
  the sequence. Decide at the milestone whether stale-branch inflation or a
  missed same-machine collision is the worse failure mode.
- Purely local: no network, no coordination, ids stay sequential and
  canonical-width. The `ids-unique` check already reads the trunk ref; this
  widens only the *allocation*-time view, not the check.

## Alternatives considered (rejected in the originating discussion)

- **Require entity creation on main via PRs.** Fatal flaw: a branch cannot
  reference a not-yet-allocated id while the work needing that reference is in
  flight. Also inverts the repo's trunk-based model and adds ceremony to a solo
  workflow.
- **Random / sparse ids (random gaps or a random suffix).** Lowers collision
  probability but never to zero, and sacrifices the contiguous, memorable,
  canonical-width id sequence (ADR-0008; the "stable readable id" commitment).
- **Per-worktree id lanes (disjoint numeric offsets).** Guarantees no inter-lane
  collision but fragments the single global sequence and needs lane assignment.
  YAGNI for a solo-plus-occasional-contributor repo.
- **Symbolic ids resolved to canonical at merge.** Solves both collisions and the
  reference problem, but adds a two-phase id model (placeholder → canonical) with
  a finalize step and placeholder ids in trailers/history, muddying the "id
  survives rename, cancel, and collision" commitment. Over-engineered for the
  observed pain.

The design goal these reject in favour of: not "eliminate collisions" (class 3 is
unpreventable with local information) but **shrink the window to truly-concurrent
cross-machine work and make that residual cheap to cure.**

## Provenance

Emerged from a design discussion on reducing merge-time id-collision risk
(2026-06-22). Primary gap of a three-gap set: sibling of `G-0273`
(fetch-before-allocate, narrows class 2) and `G-0274` (batch reallocate, cheapens
the class-3 cure). Same shared-worktree solo-dev problem family as `G-0269`
(HEAD-drift guard) and `G-0270` (epic activation on a non-trunk branch).
