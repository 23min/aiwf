---
id: G-0269
title: Mutating verbs lack a HEAD-drift guard against shared-worktree session races
status: addressed
discovered_in: E-0043
addressed_by_commit:
    - ad5b9cd0
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

## Direction

Reuse G-0270's `promote-on-wrong-branch` expected-branch derivation as a
synchronous pre-commit refusal inside `aiwf promote` itself, scoped to the two
transitions that have a well-defined "correct branch" under ADR-0010: epic
`proposed -> active` (expects trunk) and milestone `-> in_progress` (expects
the parent epic's ritual branch). Before landing the commit, check whether
`HEAD` is currently on the expected branch; refuse if not, in the same shape
as the existing G-0329 "no git operation in progress" guard at the top of
`verb.Apply` — the sovereign act is prevented from landing on the wrong
branch, rather than only flagged after the fact.

A general `--expect-head <sha>` guard (capturing `HEAD` at a ritual's
preflight and passing it forward for every mutating verb to check) and
extending `repolock`'s acquire window were both considered and rejected:
`repolock` only synchronizes aiwf-driven verbs against each other — a
concurrent plain `git checkout` by a human or another terminal never touches
it — and cannot span separate CLI invocations across a ritual's
human-deliberation gaps regardless. A general per-verb flag would touch every
mutating verb's CLI surface and every ritual skill's prose for a race that
has occurred once; the two activating-promote transitions are the only ones
with a well-defined "expected branch" today, so scoping the guard to them is
proportionate to the actual risk.

The invariant this guard upholds: **an activating-promote commit never lands
on a branch other than the one ADR-0010 expects — it refuses and re-surfaces
instead.**

## Provenance

Discovered while starting E-0043 (sibling-worktree activation). This is the
prevention half of the lesson; the post-hoc detection half is its sibling gap
G-0270 (the "epic activation on a non-trunk branch" check finding). Cousin of M-0106's
`isolation-escape` finding, which catches branch-binding drift for authorize
scopes.
