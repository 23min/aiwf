---
id: D-0073
title: Planning happens on main; implementation work happens in a worktree
status: proposed
---
> **Date:** 2026-08-22 · **Decided by:** human/peter

## Question

Both planning rituals refer to a "ritual branch" holding the planning commits,
and spend a closing step merging it to main. Neither cuts that branch. Both also
argue that planning data held on a branch is hostage — other sessions see only
main's view, parallel epics walk separate id views, and milestone branches stack
on a long-lived parent. Alongside that, each states an assumption that planning
runs in a single checkout rather than a worktree, on the grounds that worktrees
add friction without payoff.

So the rituals describe two workflows at once, and justify the one they
recommend by an argument about ergonomics rather than about what the branch
costs. Which is it?

## Decision

Planning happens on main, in the main checkout. Entities are allocated and
bodies filled there, and the work is visible to the team as soon as main is
pushed. There is no ritual branch and no merge step.

Implementation work happens in a worktree. Which kind is the operator's
per-invocation choice — the Q&A `aiwfx-start-epic` already offers, defaulting to
in-repo.

No configuration knob is added. `worktree.dir` already configures where a
worktree lands; nothing today needs an opt-out from having one.

## Reasoning

Visibility is the whole point of planning landing on main, and it is the reason
the rituals already give for merging promptly. A freshly-allocated id that
exists only on a branch is invisible to every other session and clone: they
compute their own next-free id against main's view and collide later. Cutting a
branch to hold planning creates that exposure, and the merge step exists only to
close it. Not cutting the branch closes it from the start.

This is also what the repo already does for maintainers — commit directly to
trunk, no PR ceremony — so planning-on-main is the existing model rather than a
new one.

The friction argument reaches the right place by the wrong route, and the route
matters. Stated as "worktrees add friction", it reads as a carve-out from the
worktree default, which invites re-litigation on ergonomic grounds. Stated as
"planning is not branch work", it is an exception to nothing: the worktree
default governs branch work, and planning has no branch.

## Consequences

- `aiwfx-plan-epic` and `aiwfx-plan-milestones` lose their ritual-branch
  references and their merge-to-main step, and with the step goes the
  `git checkout main` that opens it — dead once there is no branch to leave.
- Each ritual's assumption paragraph is restated in terms of visibility rather
  than friction.
- `aiwfx-plan-milestones` step 10 carries the sharpest statement of why planning
  must not sit on a branch. That argument moves into the reasoning for planning
  on main; it does not go with the step it currently justifies.
- The worktree default is untouched, and planning is not an exception to it.
