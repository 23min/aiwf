---
id: M-0212
title: Union all local refs into the allocator's cross-branch id view
status: in_progress
parent: E-0052
tdd: required
acs:
    - id: AC-1
      title: Allocator unions ids from all local refs/heads
      status: open
      tdd_phase: red
    - id: AC-2
      title: Local-refs scan degrades cleanly on odd repo states
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'Two-branch integration: sibling allocation does not collide'
      status: open
      tdd_phase: red
---
## Goal

Widen the trunk-aware allocator's cross-branch view to include every local
`refs/heads/*`, not just the working tree and the single configured trunk ref.
When a sibling git worktree (or any local branch) has a freshly-committed entity,
its id is already in the shared local refs; the allocator must read it so the
next allocation skips past it instead of colliding.

This is class 1 of G-0272's taxonomy — the dominant solo+agents collision and the
only one that is *artificially* invisible (the data is on local disk; the
allocator just doesn't look). Offline, read-only, cheap.

Only the allocator (`aiwf add`) consumes the widened id set; the `ids-unique`
trunk-collision check is deliberately left on its current working-tree-vs-trunk
basis. Folding every sibling branch into the uniqueness comparison would flag the
same entity present on two branches (e.g. a feature branch forked from main) as a
false collision — the prevention win is real and side-effect-free, the detection
change is not, so this milestone takes only the prevention half. The stable-id
model is preserved entirely.

Source: G-0272. Parent epic E-0052.

### AC-1 — Allocator unions ids from all local refs/heads

The allocator's id set unions ids reachable from every local `refs/heads/*` in
addition to the working tree and the configured trunk ref. An id that exists only
on a sibling local branch raises the allocated `max`, so the next allocation
skips it. The widened set feeds `aiwf add` (prevention) only; the `ids-unique`
trunk-collision check keeps its current working-tree-vs-trunk basis.

Evidence: a test where a fixture entity id exists only on a sibling local branch
(not the working tree, not the trunk ref) and the next `aiwf add` of that kind
allocates past it.

### AC-2 — Local-refs scan degrades cleanly on odd repo states

The local-refs scan is read-only and degrades cleanly on odd repo states — a bare
repo, a detached HEAD, zero local branches, or an unreadable ref — falling back
to the current `{working-tree + trunk ref}` behavior without erroring or blocking
the add.

Evidence: edge-case tests covering no-branches and a malformed/unreadable ref,
asserting the allocator returns a valid id and emits no error.

### AC-3 — Two-branch integration: sibling allocation does not collide

End-to-end seam: two local branches sharing one object store (the worktree
scenario) do not collide. Branch A allocates an id of some kind and commits it;
an allocation driven from branch B observes A's ref and allocates the next id,
not a duplicate.

Evidence: an integration test that drives `aiwf add` (or the verb dispatcher)
across two local branches in one repo and asserts the two ids differ.
