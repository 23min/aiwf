# Epic wrap — E-0060

**Date:** 2026-07-16
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0060-resolve-cross-branch-entity-references-at-check-and-read-time
**Merge commit:** effa528e

## Milestones delivered

- M-0259 — Add cross-branch-pending tier and collision detection to reference checks (merged d89d7b1f)
- M-0260 — Resolve and render cross-branch entity content in show and list (merged 8fc5e7de)

## Summary

Lets a branch, worktree, or session validly reference an entity minted on a different local branch or worktree, in both `aiwf check` and `aiwf show`/`aiwf list`, without waiting for a merge and without copying the entity anywhere. M-0259 widened the cross-branch view the allocator already computed into a per-id (kind, path, ref) shape and used it to classify a local-tree-miss reference as the non-blocking `cross-branch-pending` subcode instead of a hard `unresolved`, escalating to the distinct `cross-branch-collision` subcode when the same id carries divergent content across refs. M-0260 is the read-side consumer of that same view: `aiwf show`/`aiwf list` resolve and render an id's content live via `gitops.BlobReader` when it's cross-branch-known but locally absent, visibly labeled as such, declining to pick a side when the content diverges. Scope held steady through both milestones; the one mid-flight correction (D-0036) softened `cross-branch-collision` from blocking to warning severity after a worked worktree scenario showed blob-SHA divergence can't distinguish a genuine duplicate-mint collision from an ordinary unmerged edit — the blocking `ids-unique`/`trunk-collision` check still catches the genuine case once both copies land in a shared tree.

## ADRs ratified

- ADR-0030 — Extend cross-branch view to reference resolution and reads

## Decisions captured

- D-0036 — cross-branch-collision severity is non-blocking, not error

## Follow-ups carried forward

- G-0416 — Cross-branch-collision can't tell an edit from a genuine collision (git-lineage disambiguation, scoped future work should D-0036's coarse v1 severity prove insufficient)
- G-0418 — Cross-branch hit/collision scan is duplicated across 3 call sites (design-review track-for-later; the primitives are shared, only the composition recipe is triplicated)
- G-0419 — `aiwf show --area` on a cross-branch id ignores its real area (narrow flag-combination gap, design-review finding)

## Handoff

Both of ADR-0030's extension points (check-side classification, read-side resolution) are shipped. The three surviving gaps are all deliberately narrow-scope: none blocks any other in-flight work, and none has a deadline. `aiwf status`/`aiwf render --format=html` surfacing cross-branch-pending references remains an explicitly out-of-scope open question from the epic spec — a candidate follow-on epic if it turns out to matter in practice, not filed as a gap since no concrete need has surfaced yet.
