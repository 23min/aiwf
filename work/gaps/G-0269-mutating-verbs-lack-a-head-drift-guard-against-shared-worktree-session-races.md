---
id: G-0269
title: Mutating verbs lack a HEAD-drift guard against shared-worktree session races
status: open
discovered_in: E-0043
---
## Problem

A git worktree's `HEAD` is a property of the **directory**, not of the session,
process, or terminal operating in it. Every process sharing a checkout shares one
`HEAD`. A concurrent session — another agent, or a human running a quick
`git checkout` in a terminal — can switch the branch out from under a multi-step
flow (a ritual's preflight → sovereign-commit window, or any single mutating
verb) with no signal. The verb then commits against whatever branch `HEAD` now
points at, silently landing the change on the wrong branch.

Observed: while starting E-0043, a parallel session created and checked out a
feature branch in the shared primary checkout between this session's preflight
(which saw `main`) and its `aiwf promote E-0043 active`. The promote landed on the
feature branch instead of `main`. Classic time-of-check / time-of-use race; the
misplaced commit was invisible until a later `git worktree list` revealed the
wrong branch.

## Why a mechanical guard, not a workflow rule

The tempting mitigation — "always use one worktree per concurrent session" —
forces a workflow, has no chokepoint (it depends on everyone remembering, every
time), and does not even cover the session + human-terminal case. Per the kernel
principle that correctness must not depend on remembered behavior, the fix is
mechanical: make any chosen layout fail-safe. Separate worktrees remain a genuine
ergonomic convenience for parallel work, never a correctness requirement.

## Direction (candidate mechanisms, to converge at the milestone)

- Capture branch + `HEAD` SHA at verb/ritual entry and refuse the commit if
  either drifted by commit time. An explicit `--expect-head <sha>` flag lets a
  ritual pin the value across its multi-step window.
- A per-worktree "last-seen HEAD" record that warns when `HEAD` moved since the
  last `aiwf` invocation with no intervening `aiwf`-driven checkout.
- Extend `repolock` to span the sovereign-commit window.

Whichever lands, the invariant is: **a mutating verb never commits to a branch
other than the one the operator was working against — it refuses and re-surfaces
instead.**

## Provenance

Discovered while starting E-0043 (sibling-worktree activation). This is the
prevention half of the lesson; the post-hoc detection half is its sibling gap
G-0270 (the "epic activation on a non-trunk branch" check finding). Cousin of M-0106's
`isolation-escape` finding, which catches branch-binding drift for authorize
scopes.
