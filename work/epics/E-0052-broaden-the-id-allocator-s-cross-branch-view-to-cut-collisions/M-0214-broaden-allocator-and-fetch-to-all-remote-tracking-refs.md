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
      status: met
      tdd_phase: done
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

## Work log

Phase timeline is authoritative in `aiwf history M-0214/AC-<N>`; not duplicated here.
Test fixture ids in backticks are literal allocator outputs, not entity references.

### AC-1 — Allocator unions ids from all remote-tracking refs
`gitops.RemoteTrackingRefs` (lists `refs/remotes/*`, skips symbolic HEAD) →
`trunk.RemoteRefIDs` (via the shared `refIDs` helper) → `Tree.RemoteRefIDs` field +
`AllocationIDs()` (now trunk ∪ local ∪ remote); `treeload.go` stamps it. The
allocator already consumed `AllocationIDs()` (M-0212), so no `add.go` change was
needed for the scan. · commit `71d11ea8` · tests: `RemoteTrackingRefs`,
`RemoteRefIDs_*`, `Tree_AllocationIDs_UnionsTrunkLocalAndRemoteRefs`,
`TestAdd_AllocatesPastNonTrunkRemoteRef` (clone with a non-trunk remote `G-0009` →
allocates `G-0010`, not `G-0002`).

### AC-2 — aiwf add --fetch fetches all remotes
`gitops.FetchAll` (`git fetch --all`); `add.go`'s `--fetch` block now calls it. The
M-0213 single-branch chain (`FetchBranch` / `FetchTrunkBestEffort` /
`parseRemoteTrackingRef`) was removed. · commit `71d11ea8` · tests:
`TestFetchAll_*`, `TestAdd_FetchAllReflectsNonTrunkRemoteID` (post-clone non-trunk
branch brought by `--fetch` → `G-0010`; without `--fetch` → `G-0002`).

### AC-3 — Broadened fetch and remote-refs scan degrade cleanly, never block
Best-effort preserved: `add.go` warns + continues on a fetch failure; no-remote
`git fetch --all` is a clean no-op. · commit `71d11ea8` · tests:
`TestAdd_FetchBadRemote_WarnsButSucceeds` (unreachable remote → warning + exit OK),
`TestAdd_FetchBestEffort_NoRemote` (no-remote → no warning), `RemoteRefIDs`
no-remotes/not-a-repo degradation. Both load-bearing branches vacuity-checked
(disable scan → AC-1/AC-2 fail; disable fetch → AC-2/AC-3 fail).

## Decisions made during implementation

- **Full G-0316 (scan-all + fetch-all), superseding M-0213's trunk-only `--fetch`.**
  The choice between (A) adding only the remote-refs scan and (B) also broadening
  `--fetch` to `git fetch --all` was put to the operator, who chose B — the coherent
  "everything the scan reads, `--fetch` refreshes." Accepted costs: an id is taken
  the moment it is pushed anywhere (an abandoned remote branch burns its id), and the
  scan is O(remote branches) per allocation. `aiwf reallocate` stays the backstop for
  the unpushed-elsewhere residual.
- **Shared `refIDs` helper.** `LocalRefIDs` (M-0212) and `RemoteRefIDs` differ only in
  the ref-listing function, so they delegate to one `refIDs(ctx, workdir, listRefs)` —
  the rule-of-two-identical-bodies extraction, not speculative abstraction.

## Validation

- `make check-fast` (golangci-lint + go vet + full `go test` incl
  `internal/policies`): **green**.
- `go build ./...`: **green**. `golangci-lint` over the affected package trees:
  **0 issues**.
- Branch coverage: `RemoteTrackingRefs` / `FetchAll` / `RemoteRefIDs` / `LocalRefIDs`
  / `AllocationIDs` 100%; `refIDs` 92.3% — the sole uncovered line is the justified
  `//coverage:ignore` (the `for-each-ref` error branch shared by both listers).
- `aiwf check`: **0 errors**.

## Deferrals

None — this milestone **closes G-0316**.

## Reviewer notes

- **Independent two-lens review before wrap.** Code-quality (`wf-review-code`) →
  **APPROVE**: allocation-only invariant verified (the `ids-unique` check reads only
  `TrunkIDs`); clean removal (no dangling refs to the deleted M-0213 functions);
  non-vacuity confirmed; slice-aliasing safe; lock-held-across-fetch correct-by-design.
  It caught a stale `add.go` fetch-block comment (still describing M-0213's trunk
  refresh + no-remote warning) — fixed inline. Design (`wf-rethink`) on the unit →
  **KEEP**, no rewrite (the `refIDs` DRY, the parallel local/remote structure, the
  removal, `git fetch --all`, and the no-dedup-of-AllocationIDs were all judged right).
- **Folded in:** the stale `add.go` comment, generalized the `refIDs`
  `//coverage:ignore` wording (the helper is now generic over the ref-lister), and a
  one-line note on the deliberate trunk overlap in `AllocationIDs`.
- **Supersedes M-0213's trunk-only `--fetch`:** removing the single-branch chain also
  resolved M-0213's `parseRemoteTrackingRef` dup-parsing watch-item, and the M-0213
  fetch tests were reconciled (no-remote is now a clean no-op, not a warning).
- **Track-for-later (non-blocking):** if a third ref-source ever appears, collapse the
  parallel `Local*`/`Remote*` triad into one merged scan + `Tree.RefIDs` field
  (rule-of-three) rather than adding a fourth parallel copy. Deferred today.
- **Accepted limitations:** the repo lock is held across `git fetch --all` (opt-in,
  best-effort, no worse in kind than M-0213); id-burn on abandoned remote branches and
  the O(remote-branches) scan cost are the design trade-offs the operator accepted.
