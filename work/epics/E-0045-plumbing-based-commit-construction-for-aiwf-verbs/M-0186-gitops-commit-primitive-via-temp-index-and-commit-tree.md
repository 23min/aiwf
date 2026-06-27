---
id: M-0186
title: gitops commit primitive via temp-index and commit-tree
status: draft
parent: E-0045
tdd: required
---
## Goal

Retire the fragile `git stash` verb-commit isolation by building a `gitops` commit-construction primitive that constructs each verb's commit against a throwaway index — never reading or writing the live index or worktree — and retrofit every mutating verb onto it.

## Problem

`internal/gitops/gitops.go` isolates a verb's commit via `git stash push --staged` + `git commit`. The stash reverts the worktree for staged renames and collides with untracked files at the old paths, aborting into a silent half-state (G-0275, fail-loud floor already shipped). The tool's per-verb atomicity is only as robust as `git stash` on an arbitrary tree — and it isn't.

## Approach

- New `gitops` primitive: build a commit from `(parent commit, set of path→blob writes)` via `GIT_INDEX_FILE`=temp → `git read-tree`/`git update-index` → `git write-tree` → `git commit-tree` → `update-ref` HEAD. The live index and worktree are never read or written to isolate the commit.
- Reconcile only the verb's own paths into the live index post-commit so `git status` is clean for them, leaving the user's other staged changes untouched.
- Retrofit `verb.Apply` onto the primitive; delete `StashStaged` / `StashPop` and the worktree-revert path.
- **Reusable seam:** the commit-construction core is factored so the later gaps-inbox milestone wraps it without a second commit path. (An AC pins this.)
- **Validation relocation (Option C):** verb owns shape by construction (drop the per-commit shape-check); relocate gitleaks to pre-push; pre-push `aiwf check` stays authoritative.

## Reversal

Still exactly one commit per verb; "undo" is unchanged (another verb invocation / `aiwf cancel`). Only the mechanism that builds the single commit changes — no new reversal surface.

## References

G-0276 (driver), G-0275 (fail-loud floor), the G-0034 → G-0112 history (why a naive `git commit --only` revert is unsafe — do not re-propose it). ACs authored at start-milestone (contract-first).
