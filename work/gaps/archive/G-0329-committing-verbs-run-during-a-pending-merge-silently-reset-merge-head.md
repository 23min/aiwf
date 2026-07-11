---
id: G-0329
title: Committing verbs run during a pending merge silently reset MERGE_HEAD
status: addressed
discovered_in: M-0216
addressed_by_commit:
    - 08b20a0acbbf4c851f2d60dc8ee839e526447ffd
---
## Context

`aiwf render roadmap --write` is a **committing verb** — it routes through the
standard apply machinery (`internal/verb/apply.go`), which inspects the staged
index, sets aside staged changes, stages its own paths, and commits exactly one
commit.

During the M-0216 wrap, the verb was run **while a `git merge --no-ff
--no-commit` was pending** (the compressed wrap sequence: merge → render → wrap
commit). The whole merge was staged in the index, so apply's staged-handling did
a `reset` ("reset: moving to HEAD" in the reflog) that **cleared `MERGE_HEAD`**.
The result was silent and destructive:

- the intended `--no-ff` wrap became a **single-parent squash** (the milestone's
  51 trailered commits were no longer reachable from the epic — `aiwf history`
  would have lost the per-commit timeline);
- the merge content was **scattered** — only ~12 of 42 files landed in the
  commit; the rest sat uncommitted in the working tree;
- **no error** was raised at any point.

Recovery required `git reset --hard <pre-wrap-tip>` and redoing the merge with
render moved to *after* the merge commit.

## The gap

A committing verb (any apply-routed verb, `render-roadmap` being the one hit)
that runs while a merge / rebase / cherry-pick is in progress can silently break
the in-progress operation. The operator gets no signal until the history is
already malformed.

## Proposed resolution

Two layers:

1. **Verb guard (mechanical, primary):** apply should refuse to run when a git
   operation is in progress — `MERGE_HEAD`, `CHERRY_PICK_HEAD`, `REVERT_HEAD`,
   or an active `rebase-merge` / `rebase-apply` in the worktree gitdir (note:
   in a `git worktree`, these live under the per-worktree gitdir, not
   `./.git/`). The verb exits with a clear operator error ("a merge is in
   progress; complete or abort it before running `aiwf <verb>`") rather than
   resetting the operation away.

2. **Ritual ordering (advisory):** the `aiwfx-wrap-milestone` /
   `aiwfx-wrap-epic` rituals must sequence `aiwf render roadmap --write` as a
   **separate commit after** the merge commit is created — never folded into a
   `--no-commit` merge window. The current "render && git add ROADMAP.md into
   the wrap commit" phrasing also mis-implies render only writes a file; it
   commits independently, so ROADMAP.md is never part of the wrap commit.

## Acceptance sketch

- With `MERGE_HEAD` present in the (worktree-aware) gitdir, an apply-routed verb
  exits non-zero with a self-explaining error and leaves the merge state intact.
- The wrap rituals render the roadmap strictly after the merge commit.
