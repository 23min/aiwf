---
id: G-0371
title: 'wf-patch: reviewer git-state mutation can silently corrupt the commit-gate diff'
status: open
---
## Problem

During a `wf-patch` review (step 5), an independent reviewer subagent verified
a finding's vacuity by running `git stash` / `git stash pop` on the file under
review, inside the same shared working tree and index the orchestrating
session was about to commit from. A plain `stash`/`pop` restores content
**unstaged** regardless of its staged state beforehand, silently desyncing the
index from the diff that had been staged before the reviewer was dispatched.
The orchestrating session then ran `git commit` without re-verifying `git diff
--cached` immediately beforehand (trusting the staging state established
before dispatch), and the resulting commit landed missing the actual fix —
only the new pinning test, which asserted content the same commit didn't
contain. Caught only because the working tree was inspected afterward; would
otherwise have shipped as a broken intermediate commit (one that fails its own
test if checked out in isolation).

## Why it matters

This is a repeatable failure mode, not a one-off: any `wf-patch` review step
that hands Bash-capable git access to a subagent operating in the *same*
checkout as the pending commit risks this exact corruption whenever the
reviewer's verification method touches git state. aiwf's own history already
diagnosed the general lesson at a different layer — `G-0275` / `G-0276` /
`ADR-0022` found that `git stash` is a fragile, worktree-mutating primitive
unsuited to isolating transient state on a shared tree, and the fix there was
to stop touching the shared index/worktree entirely (a temp-index
`commit-tree` plumbing primitive), not to use `stash` more carefully.

## Shape (sketch)

Two coupled fixes, both likely landing in `wf-patch`'s own skill doc:

1. **Orchestrator-side backstop.** The commit-gate step (step 7) must
   re-run `git diff --cached` (or equivalent) immediately before `git
   commit`, every time, regardless of what happened between staging and the
   commit gate — never trust staging state carried across a subagent
   dispatch.
2. **Reviewer-dispatch contract.** A reviewer verifying revert/vacuity
   behavior must not mutate the shared working tree or index at all — no
   `git stash`, no in-place revert-then-restore. Read pre-edit content via
   `git show HEAD~1:<path>` (or equivalent plumbing) or work in an isolated
   throwaway worktree instead, per the precedent `ADR-0022` already
   established for aiwf's own verb-commit isolation.
