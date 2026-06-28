---
id: M-0187
title: Opt-in gaps inbox on a never-checked-out ref
status: draft
parent: E-0045
depends_on:
    - M-0186
tdd: required
---
## Goal

Add an opt-in, default-off gaps inbox that files a gap onto a dedicated never-checked-out ref via git plumbing — without touching the operator's HEAD, index, or worktree — by wrapping the M-0186 commit-construction primitive with a fetch → allocate → compare-and-swap → opt-in-push layer.

## Problem

Filing a gap mid-flight on a feature/epic worktree carries constant "collision fear": `aiwf add gap` allocates blind to sibling worktrees and unpushed trunk, so two contexts produce the same id, cured later by `aiwf reallocate`. A never-checked-out gaps ref removes the worktree-desync hazard and turns the residual cross-machine race into an immediate non-fast-forward rejection at file time.

## Approach

- Reuse M-0186's commit-construction primitive (build a commit from parent tree + one new blob) — this milestone is the second consumer that proves M1's seam is reusable.
- Target a never-checked-out `refs/aiwf/*`-class ref: fetch the ref, allocate the id against it, `commit-tree` from the ref tip's full tree + the new gap blob, `update-ref` with a compare-and-swap guard, opt-in push.
- Opt-in, default-off (`aiwf.yaml: gaps.inbox`); reversible (flip off → today's behaviour, no migration).
- Its own ADR settles the two norm departures (writing to a non-current ref; push inside a verb) — authored when this milestone starts.

## Depends on

M-0186 (the commit-construction primitive). Starts only after M1 wraps.

## References

G-0281 (driver), G-0272 / G-0273 / G-0274 (the kind-general collision cluster this complements; this is gaps-only), ADR-0001 (related: mint ids at trunk integration). ACs authored at start-milestone.
