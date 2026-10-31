---
id: G-0748
title: aiwf history lists the other entity's commits after a collision is reallocated
status: open
---
## What's missing

`aiwf history <id>` picks commits by id string:

1. It expands the queried id through the entity's `prior_ids` (`internal/cli/history/history.go:85-99`).
2. `ReadHistoryChain` (`internal/entityview/historyevent.go:117`) then searches `git log` for `aiwf-entity:` and `aiwf-prior-entity:` trailers that name any id in that chain (lines 131-150).

After a collision two entities have held the same id, so every commit naming that id matches both of them:

- the entity that kept the id lists the renumbered entity's `add` and `reallocate`;
- the renumbered entity lists the other entity's `add`.

Reproduced 2026-10-26 with aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`), git 2.51.1, on Linux, by `docs/research/id-allocation/experiments/scripts/s0_repair.sh before-merge`:

1. Two clones each add a gap, and both get `G-0002`.
2. The first clone's branch merges.
3. The second clone renumbers its gap on its branch before merging `main` in, which keeps every reference correct (`verify: 4 correct, 0 not`), and its branch merges.

In a fresh clone of `main`:

```
$ aiwf history G-0002          # the first clone's gap
… human/alpha  add         aiwf add gap G-0002 "Alpha gap"  7907cf7
… human/beta   add         aiwf add gap G-0002 "Beta gap"  3e832c8
… human/beta   reallocate  aiwf reallocate G-0002 -> G-0004  eb7fda2
$ aiwf history G-0004          # the second clone's gap
… human/alpha  add         aiwf add gap G-0002 "Alpha gap"  7907cf7
… human/beta   add         aiwf add gap G-0002 "Beta gap"  3e832c8
… human/beta   reallocate  aiwf reallocate G-0002 -> G-0004  eb7fda2
```

`aiwf show G-0002` lists the same three commits under "Recent history (3)".

## Why it matters

`aiwf history` is how aiwf answers who created an entity, when, and on whose authority. After every collision repair, each of the two entities reports the other's creation. Each appears to have been created twice, by two people under two titles, and the entity that kept its id appears to have been reallocated away from it. Anyone auditing either entity, person or agent, is handed the other entity's acts as its own.

## Relates to

Archived [G-0037](archive/G-0037-cross-branch-id-collisions-split-the-audit-trail-allocator-is-local-tree-only.md) named this symptom in its "Why this is severe" list: "`git log --grep "aiwf-entity: G-035"` returns commits from both branches under one id that now means two entities". It is marked addressed, and its resolution says the prior-id chain makes "pre-rename, rename, and post-rename commits arrive as one chronological timeline". For an id that two entities held, that timeline is the union of both entities' commits. [G-0434](G-0434-resolveviapriorids-prefers-a-stale-prior-ids-match-over-a-reused-live-id.md) is another reader that resolves the wrong entity when one id has had two holders.
