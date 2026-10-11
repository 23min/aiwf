---
id: G-0734
title: Design docs and AllocateID's comment say allocation reads only two trees
status: open
discovered_in: M-0336
---
## What's missing

Descriptions of id allocation outside `docs/design/id-allocation.md`'s update state a narrower scan than the allocator runs. `docs/design/design-decisions.md`, §"Stable ids and rename ergonomics", says "the allocator reads two trees: the caller's working tree and the configured trunk ref", and names "two feature branches that diverged before either pulled the other" as the residual collision `ids-unique` catches. The doc comment on `entity.AllocateID` (`internal/entity/allocate.go`) calls its `trunkIDs` input "id strings already present in the configured trunk ref's tree" and describes the same residual. `docs/design/id-allocation.md`'s §"Trunk-aware allocator" opens with the two-tree reading and quotes an earlier form of that doc comment, though its later update states the full scan.

Every caller passes `Tree.AllocationIDs()` as `trunkIDs` (`internal/verb/add.go`, `internal/verb/reallocate.go`, `internal/verb/import.go`), and `Tree.AllocationIDs` (`internal/tree/tree.go`) returns the trunk ids plus `LocalRefIDs` and `RemoteRefIDs`. In the aiwf devcontainer at `5b9fc3d0b`, `go test -count=1 -run TestTree_AllocationIDs ./internal/tree/` and `go test -count=1 -run TestAdd_TwoBranchesNoCollision ./internal/cli/integration/` pass: two local branches allocate different ids.

## Why it matters

The design-decisions document is the normative statement of what aiwf commits to. It names as the remaining collision two feature branches that diverged, a case the allocator now covers whenever the other branch's ref is present, and leaves unstated the one that remains: ids allocated in separate clones before either is pushed and fetched. A reader reasoning about collision risk from it, or changing `AllocateID` from its doc comment, works from the narrower model and can drop or duplicate the ref scan without seeing that it exists.
