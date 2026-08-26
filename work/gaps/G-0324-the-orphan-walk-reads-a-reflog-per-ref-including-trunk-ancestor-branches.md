---
id: G-0324
title: The orphan walk reads a reflog per ref, including trunk-ancestor branches
status: open
priority: low
discovered_in: M-0216
---
## What's missing

`WalkOrphanedAICommits` (`internal/check/reflog_walk.go`) runs
`git reflog show <ref>` once per ritual branch under `refs/heads/`, and the
isolation-escape oracle's `--all` revwalk traverses every ref alongside it.
Neither asks whether a ref is already an ancestor of trunk.

For a merged `epic/*`, `milestone/*` or `patch/*` branch that question has a
cheap answer and settles the work: every commit on it is reachable from trunk
and was validated when trunk was walked, so the reflog read buys nothing. The
walk should skip trunk-ancestor refs.

## Why it matters

Cost scales with ref count rather than with unvalidated work. The orphan walk's
per-ref reflog reads (~46 measured at filing) are paid on every `aiwf check`,
and a repository that keeps its merged ritual branches pays more of them the
longer it runs.

Keeping those branches is deliberate, not neglect. `aiwfx-wrap-epic`'s
Conventions state that *"local branches are preserved (so `tig` / `gitk` keep
labelling history); origin branches for completed milestones are deleted to
reduce remote refname clutter."* A local ref is what labels a merge in
`git log --graph`, so deleting it to save a reflog read trades a readable
history for a faster check.

The ancestry test removes that trade. Once the walk skips trunk-ancestor refs,
a preserved branch costs one `merge-base --is-ancestor` instead of a reflog
read, and the wrap ritual's preservation policy and this gap's cost concern
both hold at once.

## Resolution shape

Make the walk skip refs that are ancestors of trunk. Pruning merged branches
reduces the same cost and requires no code, but it contradicts the wrap
ritual's stated policy and has to be repeated after every wrap — it is a
workaround this fix retires, not standing advice.

M-0216/AC-6 already moved the oracle's first-parent index in-memory; the
per-ref reflog reads in the orphan walk are what remains.

G-0323 proposed a validated-trunk watermark so `aiwf check` walks only new
commits, which would have subsumed this. It is `wontfix`, so the per-ref cost
has no other resolution in flight.
