---
id: M-0270
title: Collapse duplicated verb-layer helpers onto their shared seams
status: draft
parent: E-0069
tdd: none
acs:
    - id: AC-1
      title: rename and reallocate share one path-rewrite helper with both tail behaviors
      status: open
    - id: AC-2
      title: acknowledge illegal uses gitops ancestry and existence helpers, not exec
      status: open
    - id: AC-3
      title: Cancel and Promote share one cascade guard; Cancel moves to cancel.go
      status: open
    - id: AC-4
      title: reflog walk uses gitops.LocalBranchRefs; porcelain-only fns annotated
      status: open
    - id: AC-5
      title: doctor reads hook and guidance markers via initrepo; completeHookNames deduped
      status: open
---
## Goal

Collapse the audit's mechanical duplications onto the shared seams the codebase
already owns, so each duplicated helper exists exactly once.

## Context

Findings F2/F3/F5/F7/F9/F12 of `docs/initiatives/verb-layer-cleanup.md` — all
verified, none requiring a design decision. Each item is a local fold onto an
existing exported helper (`gitops.IsAncestor`, `gitops.LocalBranchRefs`,
`initrepo`'s marker functions) or an extraction both call sites already comment
they should share.

## Acceptance criteria

### AC-1 — rename and reallocate share one path-rewrite helper with both tail behaviors

### AC-2 — acknowledge illegal uses gitops ancestry and existence helpers, not exec

### AC-3 — Cancel and Promote share one cascade guard; Cancel moves to cancel.go

### AC-4 — reflog walk uses gitops.LocalBranchRefs; porcelain-only fns annotated

### AC-5 — doctor reads hook and guidance markers via initrepo; completeHookNames deduped

## Constraints

- Pure refactors: no behavior change; existing tests stay green and each fold
  lands with a referencing test or rides one that pins the seam.
- The `dupl` tripwire (G-0423) stays green without new baseline entries.

## Design notes

- F2's shared path-rewrite helper parameterizes the "no second hyphen" branch —
  rename appends the new slug, reallocate discards and replaces; the verified
  semantic fork must survive the merge.
- F5 moves `Cancel` into its own `internal/verb/cancel.go` alongside the shared
  cascade guard.

## Out of scope

- The FinishVerb/envelope triad (its own milestone).
- The contract-gate and rewidth-sweep judgment calls (decision entities, not
  builds).

## Dependencies

- None — parallel-safe with the bug-fix milestone.

## References

- `docs/initiatives/verb-layer-cleanup.md` §F2/§F3/§F5/§F7/§F9/§F12; G-0423.

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
