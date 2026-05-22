---
id: G-0153
title: aiwf status --worktrees stale rendering misleads and elides parent epic
status: open
discovered_in: M-0124
---
## What's missing

`aiwf status --worktrees` collapses every worktree whose driver entity reaches a terminal status into a single "STALE" rendering arm that both (a) conflates two genuinely different cleanup states under one label and (b) silently drops the parent-epic breadcrumb that is the operator's primary contextual cue.

The stale arm currently emits:

    Worktree: /path/to/worktree
      ⎇ milestone/M-NNNN-...  •  last commit Xh ago
      M-NNNN — Title [done]
      STALE — driver is terminal; cleanup: git worktree remove /path/...

Two problems with that:

1. **Misleading cleanup hint.** "Driver = terminal" does not imply "safe to remove the worktree." When the branch carries ahead-of-trunk commits — i.e. the milestone has been *wrapped* on the branch but the branch has not been merged to trunk yet — running `git worktree remove` drops the checked-out state and any uncommitted edits. The commits themselves stay reachable via the branch ref, but the operator has lost the working-tree they were counting on for the wrap-and-merge step. The cleanup framing should only fire when the branch is fully merged.

2. **Parent epic elided.** When the driver milestone is terminal, the stale arm returns before `renderMilestoneDriver` runs, so the parent epic line never gets printed. But the *active* parent epic is exactly the context the operator needs to make sense of a wrapped-but-not-yet-merged milestone ("right, this is the wrap step in E-NNNN"). Terminal driver status doesn't change the structural relationship.

## The honest three-way distinction

Terminal driver status decomposes into three genuinely different operational states:

| State              | Driver status      | Ahead-of-trunk | Right hint                               |
|--------------------|--------------------|----------------|------------------------------------------|
| wrap-pending       | `done`/`addressed` | > 0            | "merge to trunk before removing"         |
| safe-to-remove     | `done`/`addressed` | 0              | "safe: `git worktree remove <path>`"     |
| abandoned          | `cancelled`/`rejected`/`wontfix` | any | "abandoned; `git worktree remove <path>`" |

The data is already on hand: `branchFirstAheadCommitTime` (in `internal/cli/status/worktrees.go`) already runs `git log main..branch` to derive `CreatedTime`; the same query class yields the ahead-count for free. A new `WorktreeView.AheadOfTrunk int` field plumbed through `BuildWorktreeViews` is the minimum surface.

## Fix shape

Three changes in `internal/cli/status/worktrees.go`, scoped tightly:

1. `BuildWorktreeViews` populates an ahead-of-trunk count on each `WorktreeView`. Best-effort: zero on git failure.
2. `renderWorktreeSection`'s stale arm branches on the new field:
   - **wrap-pending** keeps the in-flight body layout (parent epic + ACs + depends_on + surfaced gaps) but appends a `WRAP PENDING — merge to trunk first` line instead of the cleanup hint;
   - **safe-to-remove** and **abandoned** keep the compact cleanup-hint rendering but still emit the parent epic breadcrumb when `DriverKind == milestone`.
3. The JSON envelope's `WorktreeView` gains the new field (omitempty).

ACs (suggested):

- AC-1: wrap-pending worktree (driver `done`, branch ahead of trunk) renders parent epic + the body blocks + a "merge first" hint; no `git worktree remove` suggestion.
- AC-2: safe-to-remove worktree (driver `done`, branch even with trunk) renders parent epic breadcrumb + compact cleanup hint with the remove command.
- AC-3: abandoned worktree (driver `cancelled`/`rejected`/`wontfix`) renders parent epic breadcrumb + an "abandoned; safe to remove" line.
- AC-4: JSON envelope carries the new wrap-state field on every `WorktreeView`.

## Why it matters

The current rendering actively encourages the operator to run the wrong command at the wrong time. "STALE — `git worktree remove ...`" is a confident-looking suggestion; an operator who trusts the kernel will obey it. In the wrap-pending case that costs them their checked-out working tree mid-wrap — a small but real cleanup-and-restart penalty. Worse, the missing parent-epic context makes the suggestion look complete, not lossy: there's no visible reminder that this worktree's work is unmerged.

This pattern — confidently-wrong cleanup advice from the planning surface — is exactly the failure mode aiwf's "framework correctness must not depend on the LLM's behavior" rule is supposed to eliminate. The chokepoint here is the verb's output, not skill discipline.

## Test approach

The live scenario that surfaced this gap (the M-0124 wrap session) is transient — once M-0124 merges to trunk, the specific state evaporates. The bug must be pinned by a fixture-based test under `internal/cli/status/worktrees_test.go`, not by recreating the live session by hand.

Shape:

- Test fixture passes hand-constructed `WorktreeView` slices with `AheadOfTrunk` set per matrix cell directly into `RenderWorktreeViews`.
- One subtest per cell of the three-way matrix (wrap-pending, safe-to-remove, abandoned).
- Each subtest asserts both the structural rendering (parent epic line present, body blocks present iff wrap-pending) and the cleanup-hint phrasing.

The field-population (`branchAheadOfTrunkCount`) can be tested via a small synthetic-git-repo integration test, matching the patterns already used by other `internal/cli/*` tests.

## Live BEFORE evidence

Captured 2026-05-22 from this consumer repo with M-0124 wrapped (status: `done`) on its branch but the branch not yet merged to trunk. `aiwf status --worktrees` (binary built from main at 367a146b, which already includes the per-worktree-tree resolution fix):

    Worktree: /workspaces/aiwf-M-0124-positive-cell-coverage
      ⎇ milestone/M-0124-positive-cell-coverage  •  last commit 1h ago
      M-0124 — Positive cell coverage: legal workflows succeed with expected post-state [done]
      STALE — driver is terminal; cleanup: git worktree remove /workspaces/aiwf-M-0124-positive-cell-coverage

Both bugs visible: the cleanup hint suggests removing the worktree even though the branch carries unmerged commits, and the parent epic E-0033 (which is active and the wrap target) does not appear in the rendering at all.

## History

Surfaced during the M-0124 worktree's wrap step, with the parent epic E-0033 active and the M-0124 milestone freshly promoted to `done` on its branch but not yet merged to trunk. The operator caught the "STALE — `git worktree remove ...`" advice as misleading on read and also noted the missing E-0033 breadcrumb. Filed against the live scenario to preserve the before-evidence.
