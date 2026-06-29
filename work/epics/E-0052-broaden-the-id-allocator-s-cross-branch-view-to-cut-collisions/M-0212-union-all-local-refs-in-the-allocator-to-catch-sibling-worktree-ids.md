---
id: M-0212
title: Union all local refs in the allocator to catch sibling-worktree ids
status: draft
parent: E-0052
tdd: required
---
## Goal

Widen the trunk-aware allocator's cross-branch view to include every local
`refs/heads/*`, not just the working tree and the single configured trunk ref.
When a sibling git worktree (or any local branch) has a freshly-committed entity,
its id is already in the shared local refs; the allocator must read it so the
next allocation skips past it instead of colliding.

This is class 1 of G-0272's taxonomy — the dominant solo+agents collision and the
only one that is *artificially* invisible (the data is on local disk; the
allocator just doesn't look). Offline, read-only, cheap. Both the allocator
(`aiwf add`) and the `ids-unique` trunk-collision check consume the widened id
set. The stable-id model is preserved entirely.

Source: G-0272. Parent epic E-0052.
