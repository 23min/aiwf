---
id: ADR-0025
title: Allocator's cross-branch view spans all refs, fed to allocation only
status: accepted
---
## Context

`aiwf add` allocates `max(observed ids) + 1`. G-0037 shipped a *trunk-aware* allocator
that read two trees — the working tree and the single configured trunk ref
(`refs/remotes/origin/main`) — and deferred any broader scan. That view is blind to two
collision classes that bite in practice (catalogued in G-0272):

- **Local sibling branches / worktrees** — a freshly-committed id sits in the shared
  local `refs/heads/*`, but the allocator refused to look. The dominant solo + AI-agent
  case (the same operator, different worktree sessions, neither pushed).
- **Pushed-but-unmerged remote branches** — a teammate's id on a feature branch that is
  in `refs/remotes/*` but not yet on trunk.

In both, a second allocation re-uses the id, and the collision only surfaces at push via
the `ids-unique` check, to be repaired by `aiwf reallocate`. The structural fix
(ADR-0001: mint ids at trunk integration via a per-kind inbox) is heavyweight and
changes the stable-id-from-creation model; E-0052 sought the cheap, model-preserving
point on the same axis.

## Decision

Widen the allocator's read set to the **full published cross-branch view** — the working
tree, **every local `refs/heads/*`** (M-0212), **every remote-tracking `refs/remotes/*`**
(M-0214), and the configured trunk ref — so the locally-knowable collision classes are
caught at allocation time. `aiwf add --fetch` opt-in-refreshes that view with a
best-effort `git fetch --all` before allocating (M-0213 → M-0214).

The widened set feeds **allocation (prevention) only**. The `ids-unique` check keeps its
working-tree-vs-trunk basis and is *not* widened: folding sibling branches into the
uniqueness comparison would false-flag the same entity legitimately present on two
branches as a duplicate. Prevention is symmetric across branches; detection stays anchored
to trunk.

The stable-id-from-creation model is preserved entirely — no inbox, no mint, no slug
phase, no new id shape. This is deliberately the cheap, model-preserving midpoint on the
same axis as **ADR-0001** (mint at trunk integration), which remains the proposed
structural endpoint for team / sustained-parallel-agent scale. ADR-0001 is deferred by
this decision, not rejected.

## Consequences

- The dominant collision classes (local worktrees, a stale trunk view, pushed feature
  branches) are caught at allocation rather than at push — far fewer `aiwf reallocate`
  repairs in solo + agent and small-team work.
- An id is now "taken" the moment it is pushed *anywhere*, not only when it lands on
  trunk. An id consumed by a remote branch that is later abandoned is burned permanently.
  Accepted: ids are cheap and monotonic.
- The scan costs O(local + remote branches) `git ls-tree` invocations per `aiwf add`.
  Trivial at the solo / handful-of-branches scale this targets; it grows linearly, so a
  repo carrying hundreds of stale branches would pay for them on every allocation.
- The residual race is unchanged: a peer who has allocated but **not pushed** is
  unknowable locally, so `aiwf reallocate` remains the backstop for the irreducible
  cross-machine concurrent case (its batch-cure is G-0274's open domain).
- Keeping `ids-unique` trunk-anchored preserves its soundness while the allocator's view
  broadens — the two axes (prevent / detect) evolve independently.
- If team-scale parallel-agent friction ever justifies the heavier model, ADR-0001 is the
  next step on this axis, now with a shipped cheaper baseline to compare against.
