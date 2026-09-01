---
id: D-0083
title: Ship the commit-msg shipped-surface guard unscoped
status: proposed
---
> **Date:** 2026-08-31 · **Decided by:** human/peter

## Question

The commit-msg hook's shipped-surface guard runs `git diff --cached --name-only`
on every commit in every repo that installs it, to see whether the staged change
touches the ritual authoring tree. That tree exists only in the aiwf repo, so in
a consumer repo the subprocess runs, finds nothing, and can never find anything.

The same property already has a chokepoint: the CI-tier skill-edit provenance
backstop, which judges commits after the fact and is a Go policy test that never
ships. So the question is whether the earlier catch is worth a subprocess every
consumer pays for and no consumer can benefit from.

## Decision

Keep the guard in the shipped hook, unscoped.

## Reasoning

The cost is one `git diff --cached --name-only` per commit — the same call git
itself makes constantly, against an index already warm in the page cache. What
it buys, where it can fire, is the difference between a one-second refusal at
composition and an amend or an interactive rebase once the commit exists and
others sit on top of it. On a branch the CI backstop resolves its base to the
merge-base with trunk, so a forgotten trailer persists for the branch's whole
life until history is rewritten.

Scoping it to the aiwf repo was the obvious alternative and was rejected because
there is no mechanism to scope it with. The CI backstop is restricted to this
repo by being a test that never ships, which a materialized hook cannot imitate;
detecting the repo by module path or by the presence of the authoring tree would
be a new inference, and the path predicate already yields the same answer
without one — a consumer repo has no such path, so nothing it stages matches.

Dropping the composition-time catch entirely was the other alternative. It loses
the property this milestone set out to add, leaving only the backstop whose
lateness is the defect being fixed.

What would reopen this: a measurement showing the hook's latency is felt. The
decision rests on the cost being negligible, not on it being zero, and that is
an empirical claim nobody has yet had reason to test.
