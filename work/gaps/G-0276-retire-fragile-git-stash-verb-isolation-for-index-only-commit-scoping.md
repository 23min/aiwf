---
id: G-0276
title: Retire fragile git-stash verb isolation for index-only commit scoping
status: open
---
## Problem

`aiwf` isolates each mutating verb's commit by stashing the user's pre-staged
index (`git stash push --staged`, `internal/gitops/gitops.go:175-177`) and then
committing the whole index (`Commit` → plain `git commit -m`,
`internal/gitops/gitops.go:113-116`), so the commit contains exactly the verb's
entity mutation plus any hook-added files. The stash is a global, mutable,
failure-prone operation standing in for a local, surgical one: it reverts the
worktree for staged renames, which collides with untracked files at the old paths
and aborts (the cause of the `G-0275` half-state). The tool's per-verb atomicity
is only as robust as `git stash` is on an arbitrary working tree — and `git
stash` is fragile on renames, rename-with-modify, and untracked-vs-tracked path
collisions.

## History — why not just go back to pathspec commit

aiwf originally committed by pathspec (`git commit --only -- <paths>`) and
switched to the stash for `G-0034`, because pre-commit hooks that `git add` extra
files (notably the then-pre-commit STATUS.md regenerator) interacted poorly with
`--only`: git recorded the hook's addition in HEAD but reset the post-commit index
to only the named paths, leaving a phantom staged-deletion behind.

That rationale is now **stale for aiwf's own installed hooks**: `G-0112` moved
STATUS.md regeneration to the post-commit hook, which explicitly does **not** `git
add` (`internal/initrepo/initrepo.go:200`), and the installed pre-commit hook now
runs only `aiwf check --shape-only` (`initrepo.go:185`). Nothing aiwf installs
adds to the index during a verb commit anymore. A naive revert to `--only` is
still unsafe, though, because an *arbitrary consumer's* pre-commit hook (e.g. a
formatter that `git add`s) would reintroduce the phantom-deletion — so the fix is
not `--only`.

## Direction (to converge at the milestone)

Replace the stash with isolation that never reverts the worktree:

- **Index save / reset / restore (preferred):** snapshot the index, `git reset`
  (mixed — clears staged entries, worktree untouched), stage only the verb's file,
  commit (consumer hooks still fire), restore the snapshot. No worktree revert, so
  the entire `G-0275` collision class disappears.
- **Temp `GIT_INDEX_FILE` + `git commit-tree` (most robust):** build the commit
  object against a throwaway index; the live index and worktree are never read or
  written. Caveat: `commit-tree` bypasses pre-commit hooks, so verb commits would
  no longer run `aiwf check --shape-only` / gitleaks. Whether that is acceptable
  is the load-bearing decision — it turns on whether hooks *must* fire on
  aiwf-authored commits.

The milestone settles one question: must pre-commit hooks fire on verb commits?
"Yes" → index save/reset/restore; "no — the verb already computed and validated
its own mutation" → commit-tree. Either retires the stash and the `StashStaged` /
`StashPop` pair entirely.

## Reversal

The verb still produces exactly one commit; "undo" is unchanged (another verb
invocation, or `aiwf cancel`). This changes only the mechanism by which the single
commit is built, not its observable result, so no new reversal surface is
introduced.

## Provenance

Discovered wrapping a milestone in a downstream consumer repo (2026-06-22) while
dogfooding aiwf. Strategic half of a two-gap pair: `G-0275` is the immediate
safety floor (transactional, fail-loud stash) that ships first; this gap removes
the fragile primitive that makes `G-0275` necessary. Cross-references the
`G-0034` → `G-0112` history so the `--only` path is not re-proposed.
