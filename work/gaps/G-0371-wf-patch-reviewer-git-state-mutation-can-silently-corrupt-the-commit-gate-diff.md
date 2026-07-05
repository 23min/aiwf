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

## Investigation — root cause traces to `wf-vacuity`, not `wf-patch` (2026-07-05)

A grep for the literal `git stash` across every embedded ritual `SKILL.md`
comes back empty — no ritual doc prescribes it. But the unsafe pattern that
produced it is real and lives one level up: `wf-vacuity`'s mutation-probe
step says *"Mutate one thing at a time; revert between mutations. The
implementation must be back to its original state when you finish"* without
ever naming a safe revert mechanism. That silence is exactly the gap the
reviewer filled in with `git stash`/`pop` in the incident described above.

`wf-vacuity` is not `wf-patch`-only. It has two required call sites:

1. `wf-patch` step 6 — "If the patch added or changed tested logic, also run
   the test-sufficiency lens (`wf-vacuity`)... a required invocation whenever
   there are assertions to attack."
2. `wf-tdd-cycle`'s vacuity check — "Immediately after the branch-coverage
   audit, invoke `wf-vacuity`... the invocation is required, not optional."
   `wf-tdd-cycle` runs during every milestone/epic AC, not just patches — a
   far more frequent path than `wf-patch` review.

So the exposure is broader than "a `wf-patch` review step": it's any
`wf-vacuity` mutation probe run by a reviewer sharing a working tree with
pending staged/uncommitted state, regardless of which ritual dispatched it.
A fix scoped only to `wf-patch`'s skill doc would leave `wf-tdd-cycle`'s
identical exposure untouched.

## Shape (sketch)

Two coupled fixes, retargeted per the investigation above — the primary fix
belongs in `wf-vacuity` itself so it closes the hole once for every caller,
not just `wf-patch`:

1. **Primary — safe revert in `wf-vacuity`'s mutation probe.** Specify the
   revert mechanism explicitly: read and hold the original file content
   directly (or via `git show HEAD:<path>`) before mutating, then write it
   back byte-for-byte after — never a git-index-touching verb (`stash`,
   `checkout`, `restore`) that can interact with a concurrent commit-in-
   progress. This is the fix that protects `wf-patch`, `wf-tdd-cycle`, and
   any standalone `wf-vacuity` invocation alike.
2. **Secondary — orchestrator-side backstop in `wf-patch`.** As defense in
   depth for the patch-commit flow specifically, the commit-gate step (step
   7) should re-run `git diff --cached` (or equivalent) immediately before
   `git commit`, every time, regardless of what happened between staging and
   the commit gate — never trust staging state carried across a subagent
   dispatch. A naive non-empty check is insufficient (the actual incident's
   post-corruption diff was non-empty, just missing the intended fix); the
   check needs to confirm the staged content is still what was staged before
   dispatch, not merely that something is staged.
3. **Reviewer-dispatch contract (`wf-patch` and `wf-tdd-cycle` alike).** A
   reviewer verifying revert/vacuity behavior must not mutate the shared
   working tree or index at all — no `git stash`, no in-place
   revert-then-restore against the live tree. Read pre-edit content via
   `git show HEAD~1:<path>` (or equivalent plumbing) or work in an isolated
   throwaway worktree instead, per the precedent `ADR-0022` already
   established for aiwf's own verb-commit isolation.
