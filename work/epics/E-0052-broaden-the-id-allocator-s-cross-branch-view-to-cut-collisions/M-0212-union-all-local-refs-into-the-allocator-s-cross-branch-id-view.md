---
id: M-0212
title: Union all local refs into the allocator's cross-branch id view
status: done
parent: E-0052
tdd: required
acs:
    - id: AC-1
      title: Allocator unions ids from all local refs/heads
      status: met
      tdd_phase: done
    - id: AC-2
      title: Local-refs scan degrades cleanly on odd repo states
      status: met
      tdd_phase: done
    - id: AC-3
      title: 'Two-branch integration: sibling allocation does not collide'
      status: met
      tdd_phase: done
---
## Goal

Widen the trunk-aware allocator's cross-branch view to include every local
`refs/heads/*`, not just the working tree and the single configured trunk ref.
When a sibling git worktree (or any local branch) has a freshly-committed entity,
its id is already in the shared local refs; the allocator must read it so the
next allocation skips past it instead of colliding.

This is class 1 of G-0272's taxonomy — the dominant solo+agents collision and the
only one that is *artificially* invisible (the data is on local disk; the
allocator just doesn't look). Offline, read-only, cheap.

Only the allocator (`aiwf add`) consumes the widened id set; the `ids-unique`
trunk-collision check is deliberately left on its current working-tree-vs-trunk
basis. Folding every sibling branch into the uniqueness comparison would flag the
same entity present on two branches (e.g. a feature branch forked from main) as a
false collision — the prevention win is real and side-effect-free, the detection
change is not, so this milestone takes only the prevention half. The stable-id
model is preserved entirely.

Source: G-0272. Parent epic E-0052.

### AC-1 — Allocator unions ids from all local refs/heads

The allocator's id set unions ids reachable from every local `refs/heads/*` in
addition to the working tree and the configured trunk ref. An id that exists only
on a sibling local branch raises the allocated `max`, so the next allocation
skips it. The widened set feeds `aiwf add` (prevention) only; the `ids-unique`
trunk-collision check keeps its current working-tree-vs-trunk basis.

Evidence: a test where a fixture entity id exists only on a sibling local branch
(not the working tree, not the trunk ref) and the next `aiwf add` of that kind
allocates past it.

### AC-2 — Local-refs scan degrades cleanly on odd repo states

The local-refs scan is read-only and degrades cleanly on odd repo states — a bare
repo, a detached HEAD, zero local branches, or an unreadable ref — falling back
to the current `{working-tree + trunk ref}` behavior without erroring or blocking
the add.

Evidence: edge-case tests covering no-branches and a malformed/unreadable ref,
asserting the allocator returns a valid id and emits no error.

### AC-3 — Two-branch integration: sibling allocation does not collide

End-to-end seam: two local branches sharing one object store (the worktree
scenario) do not collide. Branch A allocates an id of some kind and commits it;
an allocation driven from branch B observes A's ref and allocates the next id,
not a duplicate.

Evidence: an integration test that drives `aiwf add` (or the verb dispatcher)
across two local branches in one repo and asserts the two ids differ.

## Work log

Phase timeline is authoritative in `aiwf history M-0212/AC-<N>`; not duplicated here.

### AC-1 — Allocator unions ids from all local refs/heads
`gitops.LocalBranchRefs` (lists `refs/heads/*`) → `trunk.LocalRefIDs` + shared
`idsFromPaths` (ls-tree each, extract ids) → `Tree.LocalRefIDs` field +
`AllocationIDs()` (trunk ∪ local-refs); `add.go` / `reallocate.go` call sites
switched to `AllocationIDs()`. · commit `5e2b44eb` · tests: `LocalBranchRefs`,
`LocalRefIDs_Unions*`, `Tree_AllocationIDs_*` (6 funcs across gitops/trunk/tree).

### AC-2 — Local-refs scan degrades cleanly on odd repo states
`LocalRefIDs` returns `[]string` with no error: `IsRepo` guard, skips an
unreadable ref, degrades to nil. · commit `5e2b44eb` · tests: non-repo,
no-branches, unreadable-ref (commit's tree object deleted → lists but won't
`ls-tree`), `idsFromPaths` non-entity-path skips. Both load-bearing branches
vacuity-checked by mutation.

### AC-3 — Two-branch integration: sibling allocation does not collide
End-to-end through the real `aiwf add` dispatcher; branch B forks before branch
A's add commit, so only the local-refs scan reveals A's id. · commit `5e2b44eb`
· test: `TestAdd_TwoBranchesNoCollision`, vacuity-checked (reverting to
`TrunkIDStrings()` reproduces the `G-0001` collision).

## Decisions made during implementation

- **Allocation-only widening (the E-0052 decision).** The widened local-refs set
  feeds the allocator only; the `ids-unique` check keeps its working-tree-vs-trunk
  basis. Decided in the planning conversation before start (folding sibling
  branches into the uniqueness comparison would false-flag the same entity on two
  branches). Captured in the Goal above and at the epic level — no separate ADR or
  project decision entity warranted.

## Validation

- `make check-fast` (golangci-lint + go vet + full `go test` incl
  `internal/policies`): **green**.
- `go build ./...`: **green**. `golangci-lint` over the five affected package
  trees: **0 issues**.
- Branch coverage: `LocalBranchRefs` / `AllocationIDs` / `idsFromPaths` 100%;
  `LocalRefIDs` 92.3% — the only uncovered statement is the single justified
  `//coverage:ignore` (the `LocalBranchRefs` git-exec error path, unreachable
  post-`IsRepo`; the primitive's error arm is itself tested).
- `aiwf check`: **0 errors**.

## Deferrals

None.

## Reviewer notes

- **Independent two-lens review before wrap.** Code-quality (`wf-review-code`) →
  **APPROVE**: every load-bearing claim verified by measurement (allocation-only
  separation, never-blocks contract, branch coverage, AC-3 non-vacuity);
  slice-aliasing and missed-call-site risks cleared; 0 lint issues. Design
  (`wf-rethink`) on the `LocalRefIDs`/`AllocationIDs` unit → **KEEP**, no rewrite.
- **Both non-blocking design notes folded in:** refreshed the `trunk` package doc
  to describe the local-refs surface, and added an O(local branches) `ls-tree`
  cost note to `LocalRefIDs`.
- **`reallocate.go`** adopted `AllocationIDs()` symmetrically. Its existing tests
  cover the call site and `AllocationIDs` is unit-tested, but no
  reallocate-specific local-refs *integration* test was added — the seam is
  identical to `add`'s, which AC-3 exercises end-to-end. Non-blocking.
- **CLAUDE.md** §"Id-collision resolution at merge time" updated: the allocator
  now scans local `refs/heads/*`, so the residual collision class is
  *cross-machine* (different clones), not same-machine local worktrees;
  `aiwf reallocate` remains the cross-machine backstop.
