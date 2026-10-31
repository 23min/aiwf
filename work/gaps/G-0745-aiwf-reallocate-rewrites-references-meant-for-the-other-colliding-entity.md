---
id: G-0745
title: aiwf reallocate rewrites references meant for the other colliding entity
status: open
---
## What's missing

When `aiwf reallocate` runs on a tree that holds both entities of an id collision, it renumbers the one chosen and then rewrites every reference to the old id: `rewriteReferences` (`internal/verb/reallocate.go:468`) walks every entity except the target, the colliding entity that keeps the id included, and `findProseMentions` (`:605`) does the same for body prose. References that meant the entity keeping the id are moved to the renumbered one. Nothing in the verb tells the two sets of referrers apart: `resolveReallocateTarget` (`:355`) chooses which entity to renumber by trunk ancestry, and the rewrite then applies to all of them. A misdirected reference still resolves, so `aiwf check` reports no error.

A tree holds both entities in two situations: on a default branch after two merge requests that allocated the same id have merged, and on a branch after the default branch is merged into it, which is what the trunk-collision hint leads to. Reallocating on a branch before merging the default branch in gives the right result.

Reproduced 2026-10-25 with aiwf built from `1e8e17e6c`, git 2.51.1, Linux, on the branch path with every git hook installed. A bare repository holds an `aiwf init` tree with an epic, a milestone and one gap. Two clones are made before either does any work. The first adds a gap on the default branch, cites it from the existing gap with `aiwf edit-body`, and pushes. The second, on a branch, adds a gap, which takes the same id, and cites it from the milestone; its pre-push check then reports the collision, and it merges the default branch in and reallocates:

```
$ aiwf check
work/gaps/G-0002-beta-s-gap.md:2: error ids-unique/trunk-collision: id "G-0002" is allocated on this branch (…) and on tr…
$ git merge --no-edit origin/main
$ aiwf reallocate G-0002
aiwf reallocate G-0002 -> G-0003
$ grep -o 'See G-000[0-9] for Alpha' work/gaps/G-0001-*.md
See G-0003 for Alpha
$ grep -o 'Blocked on G-000[0-9], Beta' work/epics/*/M-0001-*.md
Blocked on G-0003, Beta
$ aiwf check
2 findings (0 errors, 2 warnings)
$ git push origin feature/B          # the pre-push hook runs aiwf check
```

The push succeeds. The tiebreaker renumbered the second clone's gap, as intended, and the first clone's citation of its own gap now names the second's.

Frontmatter references move the same way. In the same environment, two clones each added an epic, which both took the same id, and the first added a milestone under its epic; both branches merged into the default branch with `git merge --no-ff`. Reallocating the second clone's epic left the first clone's milestone in its own epic's directory with its parent pointing at the other epic:

```
$ aiwf reallocate work/epics/E-0002-beta-epic/epic.md
aiwf reallocate E-0002 -> E-0003
$ grep -H '^parent' work/epics/*/M-0002-*.md
work/epics/E-0002-alpha-epic/M-0002-alpha-s-milestone.md:parent: E-0003
$ aiwf check
4 findings (0 errors, 4 warnings)
```

Suppose two branches have each lost a collision to the same winner, and each reallocates after merging the default branch in. Each rewrites the winner's referrers to point at its own renumbered entity, so the two rewrites differ. Reproduced 2026-10-26 by `docs/research/id-allocation/experiments/scripts/a_run.sh wrong seq all`, in the same environment. Three clones collide on a gap, an epic and a milestone, and a simulated host merges a branch only when it is up to date with the default branch and `aiwf check` passes on it. Both losers reallocated in the first round, before either loser's rewrite had reached the default branch:

```
round 1: MR mr/alpha -> MERGED
round 1: MR mr/beta -> REFUSED-not-up-to-date
  beta updated: 3 reallocate(s); now G-0003 / E-0003
round 1: MR mr/gamma -> REFUSED-not-up-to-date
  gamma updated: 3 reallocate(s); now G-0004 / E-0004
round 2: MR mr/beta -> MERGED
round 2: MR mr/gamma -> REFUSED-not-up-to-date
CONFLICT (content): Merge conflict in work/epics/E-0002-alpha-epic/M-0002-alpha-ms-parent-means-alpha-epic.md
CONFLICT (content): Merge conflict in work/gaps/G-0001-seed-gap.md
```

The conflicting hunks in the third clone are the winner's two referrers, each rewritten to a different loser: `parent: E-0004` against `parent: E-0003`, and `See G-0004` against `See G-0003`. Keeping either side leaves the winner's referrer pointing at a losing entity. On the default branch, those two referrers already point at the second clone's entities.

Within one clone and its linked worktrees the collision does not arise: a branch cut after another branch allocated an id is handed the next number.

## Why it matters

The verb that resolves a collision corrupts the references it rewrites, and the result passes every check, including the pre-push hook.

A milestone can end up filed under another epic, a `depends_on` or `discovered_in` can name the wrong entity, `aiwf show` lists the wrong referrers, and a sentence citing one gap names another. Every later reader, person or agent, follows those references to the wrong entity.

The branch path above is the repair aiwf itself points to when a collision between machines is caught at push, which is the class of collision `aiwf reallocate` exists for.

When more than one branch loses, the hint's route also leaves the later losers with content conflicts in the winner's referrers, and neither side of the conflict is right.

## Relates to

The [id-lifecycle initiative](../../docs/initiatives/id-lifecycle.md)'s formal model specifies the renumber step as rewriting every reference, so that "no `refs` set anywhere contains `old`"; the behaviour above satisfies that property.
