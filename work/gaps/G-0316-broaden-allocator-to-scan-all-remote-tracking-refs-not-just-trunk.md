---
id: G-0316
title: Broaden allocator to scan all remote-tracking refs, not just trunk
status: open
discovered_in: M-0213
---
## Problem

The id allocator's cross-branch view is `{working tree + all local refs/heads + the
single configured trunk ref}`. After M-0212 it scans *every local branch*, but on
the remote side it reads only the trunk ref (`refs/remotes/origin/main`) — never the
other `refs/remotes/origin/*`. So a teammate's entity allocated on a **pushed but
not-yet-merged** feature branch is invisible to the allocator: a second clone
allocating from the same trunk base hands back the same id, and the collision only
surfaces at merge (resolved by `aiwf reallocate`).

This is an asymmetry: locally we take the "all branches" view, remotely we take the
"trunk only" view. Closing it would make the allocator's published view symmetric —
*everything published on the shared remote*, not just trunk.

## Proposed change

Two coupled levers (one without the other buys nothing — a fetch of refs the
allocator never reads is wasted, and a scan of refs that were never fetched is stale):

1. **Fetch all of origin's branches** (`git fetch origin`, not the single-branch
   fetch M-0213 added) so every `refs/remotes/origin/*` is fresh.
2. **Widen the allocator** to union ids from every `refs/remotes/origin/*`, not just
   the configured trunk ref.

For *allocation* this is always safe (it only raises `max`). It is deliberately NOT
extended to the `ids-unique` check, for the same reason M-0212 kept that check on its
working-tree-vs-trunk basis (folding sibling branches into the uniqueness comparison
false-flags the same entity present on two branches).

## Why deferred (not done in M-0213)

M-0213 scoped the fetch to the trunk ref only ("only that ref, not a full
`fetch --all`") on purpose, and this gap records the path not taken:

- **Trunk-based workflow.** This repo commits directly to trunk; feature branches are
  short-lived, so the window where a published feature branch carries unmerged ids is
  small and trunk catches up fast. The broader scan's value scales with long-lived
  feature branches, which this repo does not use.
- **Id burn.** It shifts "an id is taken" from *landed on trunk* to *pushed anywhere*.
  An id consumed by a feature branch that is later abandoned is burned permanently.
- **Cost.** O(remote branches) `ls-tree` per `aiwf add`; a repo carrying many stale
  remote branches pays for all of them on every allocation. Would likely want a
  cheaper batch scan than the per-ref `ls-tree` M-0212 uses.
- **YAGNI.** The friction that drove E-0052 was solo + agents in local worktrees
  (M-0212's domain). Teammates racing on long-lived remote branches has not bitten.
- It still does not *close* the race — a teammate who allocated but has not pushed is
  invisible regardless, so `aiwf reallocate` remains the backstop either way.

## Relationship to other entities

- **M-0212 / M-0213 (E-0052):** this is the consistent remote-side extension of
  M-0212's all-local-branches scan; M-0213 refreshes only the trunk ref.
- **ADR-0001 (mint ids at trunk integration):** the heavyweight structural endpoint
  for the team case. This gap is a cheaper, model-preserving midpoint on the same axis
  — if the team case shows real friction, weigh this against ratifying ADR-0001.
- Milestone-sized if taken on (new ACs for the remote-refs scan + the broadened
  fetch, plus a likely batch-scan optimization), not a one-commit patch.
