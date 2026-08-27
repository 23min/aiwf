---
id: G-0324
title: The orphan walk reads a reflog per ref, including trunk-ancestor branches
status: wontfix
priority: low
discovered_in: M-0216
---
## What's missing

`WalkOrphanedAICommits` (`internal/check/reflog_walk.go`) calls `reflogEntries`
once per ritual ref returned by `listRitualHeads`, so the walk spawns one
`git reflog show` process per branch — 46 at filing.

`git log -g --all` walks every reflog in one invocation, and `%gD` carries the
full ref name, so each entry still attributes to its branch:

```
caa4fbb…  refs/heads/epic/E-0001-x@{2}
74f489b…  refs/heads/main@{1}
```

Filter by ref name to the same ritual-plus-trunk set `listRitualHeads` already
computes, group in memory, and walk consecutive pairs exactly as today. Same
findings, roughly 45 fewer subprocesses.

One wrinkle to settle when it is built: `%gD` renders either the reflog index or
the date depending on `--date`, and the finding message wants the date. That is
either two batched invocations or parsing the date out of `%gD` under
`--date=iso`.

The same shape has already been paid down once in this function. M-0216/AC-1
replaced the per-pair `git merge-base --is-ancestor` fan-out — 683 subprocesses
on this repository at the M-0215 baseline — with the in-memory `CommitDAG`. The
per-ref reflog reads are what remains of that pattern.

**Skipping refs is not the lever.** A merged ritual branch cannot be skipped on
the grounds that trunk already validated it: this walk's subject is commits that
a non-fast-forward update removed from the branch, and trunk-reachability of the
branch says nothing about them. Measured in a throwaway repository — an
`epic/E-0001-x` fully merged into `main` (`merge-base --is-ancestor` yes) still
carried an orphaned `ai/` commit that was not reachable from `main`. Skipping
trunk-ancestor refs would drop that finding, and would drop it precisely on the
long-lived merged branches such a rule would target.

The `--all` traversals carry no per-ref cost to remove. `BuildCommitDAG`
(`internal/check/orphan_dag.go`) is a single `git rev-list --all --reflog
--parents`, and the bulk revwalk (`internal/gitops/revwalk.go`) a single
`git log --all`. Adding a merged ref as a tip costs one further object parse in
a traversal that marks each commit once. (Reasoned from `rev-list`'s traversal,
not measured.)

## Why it matters

Process spawns dominate the walk's cost, and they scale with the number of ritual
branches a repository keeps rather than with the amount of unvalidated work.

Keeping merged branches is deliberate. `aiwfx-wrap-epic`'s Principles state that
*"local branches are preserved (so `tig` / `gitk` keep labelling history); origin
branches for completed milestones are deleted to reduce remote refname clutter."*
A local ref is what labels a merge in `git log --graph`. Batching the reflog read
removes the cost without touching that policy, so the two hold together.

## Resolution shape

Replace the per-ref `git reflog show` fan-out with one `git log -g --all`,
filtered and grouped in memory as above. Behavior is unchanged: same refs, same
consecutive-pair walk, same findings.

`aiwf check` in this checkout no longer reproduces the 46 — one local branch, no
ritual refs — because the merged branches were pruned in the interim. Pruning
reduces the same cost and needs no code, but it has to be repeated after every
wrap and it contradicts the wrap ritual's preservation policy, so it is a
workaround rather than the resolution.

G-0323 proposed a validated-trunk watermark so `aiwf check` walks only new
commits, which would have subsumed this. It is `wontfix`, so the per-ref cost has
no other resolution in flight.

G-0644 records a correctness defect in the same function: the walk never asks
whether the older tip is reachable from trunk, so a rewound-after-merge branch
produces a false orphan. Same ten lines, different failure — that one changes
what the walk reports, this one changes what it spends.
