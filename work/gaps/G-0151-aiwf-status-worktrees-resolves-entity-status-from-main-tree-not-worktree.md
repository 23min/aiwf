---
id: G-0151
title: aiwf status --worktrees resolves entity status from main tree, not worktree
status: open
discovered_in: M-0124
---
## What's missing

`aiwf status --worktrees` (and the `Worktrees` field in `aiwf status --format=json` output) loaded a single entity tree from the main checkout's path and used it for every worktree section's driver / parent-epic / depends_on / ACs / surfaced-gaps lookups. The displayed statuses reflected the main tree's view, not the worktree branch's view of the same entities.

Concrete repro: a worktree on `milestone/M-NNNN-...` with `M-NNNN` promoted to `in_progress` on the branch but still `draft` on main rendered as `[draft]` in the driver row. The view actively misled the operator about the work they were doing.

## Why it matters

The worktree section's whole purpose is to answer "what is in flight on each worktree right now." Reading state from main inverts that — the more decoupled a branch is from main, the more wrong the section becomes. ACs marked `met` on the branch surfaced as `open`; depends_on milestones promoted on the branch still showed as `draft`. The operator had to mentally subtract a delay or cross-check by hand, which defeats the surface.

The fix is mechanical: load each worktree's own tree from its path and use it for driver-side lookups. depends_on entries that are missing on the worktree branch (e.g. added on main after the worktree branched) fall back to the main tree — sibling-worktree branches may never merge, so main is the only safe public reference. `OtherInFlight` continues to use the main tree by design (it's the trunk's view of work not driven by any worktree).
