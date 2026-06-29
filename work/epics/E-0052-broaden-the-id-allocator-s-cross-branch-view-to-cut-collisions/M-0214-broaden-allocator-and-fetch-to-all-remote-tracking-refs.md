---
id: M-0214
title: Broaden allocator and --fetch to all remote-tracking refs
status: in_progress
parent: E-0052
tdd: required
acs:
    - id: AC-1
      title: Allocator unions ids from all remote-tracking refs
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf add --fetch fetches all remotes, not just the trunk branch
      status: met
      tdd_phase: done
    - id: AC-3
      title: Broadened fetch and remote-refs scan degrade cleanly, never block
      status: open
      tdd_phase: red
---
## Goal

Complete G-0316: broaden the allocator's published cross-branch view from `{working
tree + all local refs + the single trunk ref}` to also union *every* remote-tracking
ref (`refs/remotes/*`), and broaden `aiwf add --fetch` from the single-branch trunk
refresh M-0213 shipped to a full `git fetch --all`. So an entity allocated on a
teammate's pushed-but-not-yet-merged branch is seen at allocation time instead of
colliding at merge.

This is the remote-side mirror of M-0212's all-local-branches scan: locally the
allocator already takes the "every branch" view; this milestone makes the remote side
symmetric. As with M-0212, the widened set feeds the allocator ONLY — never the
`ids-unique` check, which keeps its working-tree-vs-trunk basis (folding sibling
branches into the uniqueness comparison would false-flag the same entity present on
two branches).

It supersedes M-0213's deliberately-conservative trunk-only `--fetch` (which fetched
"only that ref, not a full `fetch --all`" — the first cut; G-0316 reconsidered it).
Accepted costs: an id is now "taken" the moment it is pushed anywhere, so an abandoned
remote branch burns its id permanently, and the scan is O(remote branches) per
allocation. `aiwf reallocate` remains the backstop for the residual race — a teammate
who has allocated but not pushed is still invisible.

Closes G-0316. Parent epic E-0052.

### AC-1 — Allocator unions ids from all remote-tracking refs

The allocator's id set unions ids reachable from every remote-tracking ref
(`refs/remotes/*`) in addition to the working tree, all local `refs/heads/*`
(M-0212), and the configured trunk ref. An id that exists only on a non-trunk
remote-tracking ref raises the allocated `max`, so the next allocation skips it. The
widened set feeds `aiwf add` (prevention) only; the `ids-unique` trunk-collision check
keeps its current working-tree-vs-trunk basis.

Evidence: a test where a fixture entity id exists only on a non-trunk remote-tracking
ref (not the working tree, not a local branch, not the trunk ref) and the next
`aiwf add` of that kind allocates past it.

### AC-2 — aiwf add --fetch fetches all remotes, not just the trunk branch

`aiwf add <kind> --fetch` refreshes *all* remote-tracking refs via `git fetch --all`
(broadening M-0213's single-branch `git fetch <remote> <branch>`) before computing
`max`, so an id that landed on any remote branch since the last local fetch is seen by
the AC-1 scan and skipped.

Evidence: a test with a local clone whose *non-trunk* remote branch is advanced
out-of-band — the `--fetch` allocation reflects the upstream id; the same allocation
without `--fetch` does not.

### AC-3 — Broadened fetch and remote-refs scan degrade cleanly, never block

Best-effort is preserved across the broadening: a `git fetch --all` failure (offline,
an unreachable remote) degrades to local-only allocation with a warning and a success
exit — never blocks the add. A repo with no remotes is a clean no-op (git exits 0), so
`--fetch` neither warns nor blocks there. The remote-refs scan is read-only and
degrades cleanly on odd repo states (no remotes, an unreadable ref) — falling back to
the current behavior without erroring.

Evidence: a no-remote repo where `aiwf add --fetch` succeeds with no warning (the
fetch-all no-op) and allocates against the local view; an unreachable-remote repo
where `--fetch` warns and still succeeds (never blocks); and an edge-case test for the
remote-refs scan on a repo with no remote-tracking refs.
