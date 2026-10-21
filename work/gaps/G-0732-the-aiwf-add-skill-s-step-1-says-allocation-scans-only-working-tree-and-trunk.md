---
id: G-0732
title: The aiwf-add skill's step 1 says allocation scans only working tree and trunk
status: open
discovered_in: M-0336
---
## What's missing

The `aiwf-add` skill states two different scopes for id allocation. Its "What aiwf does" step 1 (`internal/skills/embedded/aiwf-add/SKILL.md`, line 95) says `aiwf add` "allocates the next free id by scanning the working tree and the configured trunk ref". Its "Allocating ids across branches and clones" section in the same file (lines 122–123) says "Allocation scans the working tree, every local branch, every remote-tracking ref, and the configured trunk ref".

The code runs the second. `internal/verb/add.go:140` allocates with `entity.AllocateID(kind, t.Entities, t.AllocationIDs())`, and `Tree.AllocationIDs` (`internal/tree/tree.go`) returns the trunk ids plus `LocalRefIDs` and `RemoteRefIDs`. In the aiwf devcontainer at `8675758af`, `go test -count=1 -run TestTree_AllocationIDs ./internal/tree/` passes, and so does `go test -count=1 -run TestAdd_TwoBranchesNoCollision ./internal/cli/integration/`, which allocates on two local branches and gets two different ids.

## Why it matters

Step 1 is the account of what the verb does, and it understates the scan: an operator reading it believes `aiwf add` will hand out an id already held on a sibling or fetched branch, and coordinates ids across branches by hand where the allocator already keeps them apart. The skill ships to every repository that installs aiwf, so its two sentences disagree there too, and a reader has no way to tell which one describes the verb.
