---
id: G-0688
title: aiwfx-wrap-epic step 5 never fast-forwards local main when a worktree holds it
status: open
---
## What's missing

`aiwfx-wrap-epic` step 5, item 1, says it fetches and fast-forwards local
mainline, driving the worktree that holds mainline by path. Its block does not:
when `TARGET_WT` is set it runs `git merge --ff-only "origin/$TARGET"` with no
path, so the merge runs in the epic worktree the step is run from and acts on the
epic branch, and local mainline stays where it was. Item 2's
`git merge-base --is-ancestor main epic/E-NNNN-<slug>` then reads that stale
local branch. Item 1 also tells the operator to read its first command by exit
code, describing exit 128 as divergence and exit 0 as "does nothing", and names
no action for either.

Measured 2026-09-17 in the devcontainer with git 2.54.0. The step's block was
copied verbatim into a throwaway repo — a bare origin, `main` checked out in the
main checkout, the epic branch in `.claude/worktrees/epic/<name>`, and
`origin/main` advanced by a push from a second clone — and run from the epic
worktree. The expectation in both cases was local `main` fast-forwarded to
`origin/main` and the epic branch untouched.

- Epic branch with a commit of its own: the merge printed
  `fatal: Not possible to fast-forward, aborting.` and exited 128. Local `main`
  stayed on the base commit while `origin/main` was one commit ahead. Item 2's
  check then exited 0 against local `main`; the same check against
  `origin/main` exited 1.
- Epic branch with no commit of its own: the merge fast-forwarded the epic
  branch to `origin/main` and exited 0, where item 1 says exit 0 does nothing.
  Local `main` did not move.

## Why it matters

Whenever another clone has pushed to mainline and the epic branch has commits of
its own, item 2 reports the epic branch as up to date, so the wrap goes on to
commit the changelog entry and wrap artefact (step 6) and promote the epic to
done (step 7) on the epic branch. Only step 9, which fast-forwards local
mainline from its own worktree, finds mainline ahead and sends the operator back
to step 5. The integration of mainline, and any merge conflict or failing gate
it produces, then arrives after the epic is already `done` — the order step 5
exists to prevent.
