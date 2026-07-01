---
id: G-0332
title: status --worktrees shows a milestone-focused view from an epic worktree
status: open
discovered_in: E-0048
---
## Problem

`aiwf status --worktrees` keys each worktree's rendering off the
**checked-out branch name**, not the **worktree directory**. So a single
epic worktree renders at two different altitudes depending on which branch
happens to be checked out inside it:

- With the epic branch checked out (`epic/E-0048-…`), the driver resolves to
  the epic and the section renders the full epic view — every milestone with
  its status, plus every gap surfaced across the whole epic
  (`epicExpansion`).
- With a milestone branch checked out (`milestone/M-0197-…`) **in the same
  worktree directory**, the driver resolves to that milestone and the section
  collapses to a milestone-focused block — parent-epic breadcrumb, that
  milestone's ACs, and only *its* surfaced gaps (`renderMilestoneDriver`).

The switch happens in `correlateBranchToEntity`
(`internal/cli/status/worktrees.go`), which calls
`branchparse.ParseEntityFromBranch(branch)` as the primary signal. The
worktree's own directory path — which encodes `epic/E-0048` and is a stable
signal for "this is the E-0048 epic worktree" — is never consulted for the
driver decision.

## Why this is a problem

The design assumed **one worktree per entity**: a worktree named
`epic/E-NNNN-…` drives the epic, a worktree named `milestone/M-NNNN-…` drives
the milestone. Making the branch name the primary driver signal was a
deliberate fix (archived G-0154, where a merge's trailer cascade had
mislabeled an epic worktree as a milestone).

But a common real workflow is **one epic worktree, rotating milestone
branches inside it** — you check out `milestone/M-0197-…` in the E-0048 epic
worktree, do the work, move to the next milestone branch, all in the same
directory. Under that workflow the epic-altitude view is exactly what the
operator wants at all times (progress across every milestone; every gap the
epic surfaced), but `--worktrees` shows it only in the window when the epic
branch itself is checked out — which, during active milestone work, is almost
never.

## Desired behavior

When the worktree *directory* is an epic worktree (path parses to
`epic/E-NNNN-…`), render the full epic view — all milestones with their
statuses plus all epic-surfaced gaps — regardless of which milestone branch
is currently checked out. When a milestone branch *is* checked out, keep a
marker for it (e.g. flag the current milestone row with `→ (checked out)` /
its `in_progress` status) so the operator still sees which one they're on,
without losing the whole-epic altitude.

The signal to do this already exists and is distinct from the checked-out
branch: the worktree directory path encodes the epic id, so a fix has a clean
anchor (parse the epic from the worktree path, render `epicExpansion`, overlay
the checked-out milestone as a marker).

## Relationship to neighbours

This is a **read-side display** issue about view *altitude* (epic vs
milestone), distinct from the rest of the shared-worktree family:

- G-0277 is the sibling read-side display gap, but about **staleness** (the
  default `aiwf status` printing a stale milestone status from the wrong tree),
  not the `--worktrees` view shape. Cross-reference it.
- G-0154 (archived, addressed) is the *opposite* symptom; its fix
  (branch-name parse is the primary driver signal) is what produces the
  behaviour reported here.
- G-0269 / G-0270 cover the **mutation** side (a verb landing on the wrong
  branch); G-0157 covers perf (batching the worktree git fan-out). None touch
  the altitude question.
