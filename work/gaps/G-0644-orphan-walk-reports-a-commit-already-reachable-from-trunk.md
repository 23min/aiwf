---
id: G-0644
title: Orphan walk reports a commit already reachable from trunk
status: open
priority: medium
---
## What's missing

`WalkOrphanedAICommits` (`internal/check/reflog_walk.go`) decides a reflog entry
is orphaned by asking whether the older tip is an ancestor of the newer one:

```go
if isAncestor(older.SHA, newer.SHA) {
    continue // fast-forward; not orphaned
}
```

It never asks whether the older tip is reachable from trunk. When it is, the AI
commit landed and is auditable from the mainline — there is no orphan, and the
finding is a false positive.

Measured in a throwaway repository:

- `epic/E-0002-y` carries two `ai/` commits, merged to `main`, then the branch is
  moved back with `git reset --hard HEAD~2`.
- older tip an ancestor of newer tip → **no**, so the walk reports an orphan.
- older tip reachable from `main` → **yes**, so the commit is on the mainline and
  fully auditable.

The shape is ordinary: reusing a ritual branch after its work has merged. A
long-lived `epic/*` branch that is reset or rewound after a merge produces one
false finding per rewind.

## Why it matters

The finding exists to surface AI commits that a non-fast-forward update removed
from every ref — work that landed nowhere and that no audit reaches. A commit
already on trunk is the opposite of that case, and reporting it teaches the
reader that the finding does not mean what it says.

An operator who meets one of these has no way to tell it from a real orphan
without checking trunk-reachability by hand, which is the check's own job.

## Resolution shape

Filter the orphan by trunk-reachability before reporting: an older tip reachable
from the trunk head is not orphaned.

The comparison is free. `reflog_walk.go` already holds a `*CommitDAG` built by a
single `git rev-list --all --reflog --parents` (`internal/check/orphan_dag.go`),
and the walk already calls `dag.isAncestor` for the newer-tip comparison in the
same loop. The trunk head is already resolved — `listRitualHeads` takes it as
`trunkShort`.

Separate from the per-ref reflog cost recorded in G-0324. Both land in the same
function; this one changes what the walk reports, that one changes what it
spends.
