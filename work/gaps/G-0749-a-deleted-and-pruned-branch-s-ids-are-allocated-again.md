---
id: G-0749
title: A deleted and pruned branch's ids are allocated again
status: open
priority: low
---
## What's missing

`aiwf add` treats an id as taken only while it is visible in some ref the clone can see. `entity.AllocateID` (`internal/entity/allocate.go:59`) takes the highest number among the working tree's entities and `Tree.AllocationIDs` (`internal/tree/tree.go:240`). `Tree.AllocationIDs` holds the trunk's ids plus those on local branches and remote-tracking refs.

Suppose a branch that allocated an id is deleted on the remote, and a clone then prunes its remote-tracking refs. That id is no longer visible to the clone. If it was the highest id of its kind the clone could see, the clone's next `aiwf add` hands it out again. ADR-0025 accepts that "an id consumed by a remote branch that is later abandoned is burned permanently". That holds while the abandoned branch stays on the remote and in clones' remote-tracking refs. It stops holding once the branch is deleted and pruned.

Reproduced 2026-10-26 with aiwf built from `1e8e17e6c` (it reports `v0.42.0+dirty`), git 2.51.1, on Linux, by `docs/research/id-allocation/experiments/scripts/s0_visibility.sh`. Alpha allocates a gap on a branch and pushes it, and beta fetches. In the pruned row, alpha then deletes the remote branch, and beta runs `git fetch --prune` and `aiwf add gap`:

```
pushed-fetch    alpha:push              beta:fetch                     -> beta got G-0003
pushed-pruned   alpha:push-then-delete  beta:fetch-prune-after-delete  -> beta got G-0002
```

## Why it matters

An id that has been pushed has usually been seen by now: cited in a merge request, a review comment, a ticket, a conversation with an agent, or on another branch. Once the id goes to a new entity, every one of those citations names the new entity instead of the abandoned one, and nothing marks the citation stale.
