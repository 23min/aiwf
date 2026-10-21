---
id: G-0733
title: The aiwf-add skill says body-prose-id resolves only the working tree and trunk
status: open
discovered_in: M-0336
---
## What's missing

The `aiwf-add` skill (`internal/skills/embedded/aiwf-add/SKILL.md`, lines 141–144) says an entity on an unmerged branch "can't be referenced by id in another branch's prose until it reaches trunk", because "the `body-prose-id` check resolves only against the working tree and trunk".

`body-prose-id` resolves sibling branches too. Its index (`internal/check/body_prose_id.go`, `BodyProseIndex.CrossBranch`) is consulted after the working tree and trunk miss: an id published on another branch is reported as the non-blocking `cross-branch-pending` warning, and an id held on local branch refs alone is reported at error severity (`cross-branch-local-only`, ADR-0041) when the repository has remote-tracking refs, and as a warning when it has none. In the aiwf devcontainer at `8675758af`, `go test -count=1 -run TestBodyProseID_CrossBranchPendingTier_M0259AC2 ./internal/check/` passes.

## Why it matters

An operator following the skill backticks a valid citation of an entity a teammate has already pushed and they have fetched, or moves allocation to trunk, to avoid a block that does not occur. The skill states as one rule what the check splits by whether the id is published, so an operator cannot tell from it which citations will block a push and which will only warn.
