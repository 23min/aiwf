---
id: G-0527
title: Worktree creation is a verb; teardown is plain git, and the absence is silent
status: open
priority: low
---
## What's missing

`aiwf worktree` carries one subverb, `add`. It creates the worktree and materializes rituals into it atomically, and the repo's own guidance routes every branch-work setup through it. Nothing removes one: teardown is `git worktree remove` plus `git branch -d`, unmentioned by any surface.

The absence is silent rather than loud. Asking for the verb that does not exist reports success:

    aiwf worktree remove <path>   ->  exit 0, prints the parent help

So an operator or agent that checks the exit code reads a no-op as a completed teardown, and only discovers otherwise when a later `git branch -d` refuses because the worktree still holds the branch. That exit-code behavior is general to every parent verb rather than specific to this one — a bogus top-level verb correctly exits 2, while a bogus subverb under `contract`, `milestone`, `acknowledge` or `worktree` exits 0 — and it is the separate, wider defect.

## Why it matters

The kernel's own design rule asks what verb undoes a verb, and lists three acceptable answers: another invocation, an explicit terminal transition, or "you can't, deliberately — here's why", written down. `worktree add` currently has a fourth, unwritten answer: plain git, discovered by trying.

The asymmetry is what makes the question live. `add` exists because a bare `git worktree add` leaves the rituals absent with no warning, so the verb buys atomic materialization. Removal has no equivalent debt — the materialized artifacts are gitignored and die with the directory — so a `remove` verb might buy nothing, and "plain git, deliberately" may well be the right answer. What is wrong today is that neither answer is recorded.

## Resolution shape

Decide which of the three sanctioned answers applies, and write it down where an operator looks:

- **Plain git, deliberately.** Likely correct on the merits, since removal carries none of the materialization debt creation does. Costs a sentence in the guidance that routes people to `worktree add`, naming the teardown pair.
- **A `remove` subverb.** Buys a symmetric surface and a single documented teardown, at the cost of a verb that wraps two git commands and must then handle a dirty worktree, an unmerged branch, and a worktree holding the branch it would delete.

Either way the silent-success behavior should stop, though that belongs to the wider parent-verb exit-code defect rather than to this decision — a teardown answer recorded here is worth having whichever way that one lands.
