---
id: M-0222
title: Path-scoped single-entity history with bloom-filter maintenance
status: draft
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: single-entity history resolves via path-scoped git log over the entity path set
      status: deferred
      tdd_phase: red
    - id: AC-2
      title: aiwf update maintains changed-path bloom filters idempotently
      status: deferred
      tdd_phase: red
    - id: AC-3
      title: path-scoped history equals trailer-grep history including renamed entities
      status: deferred
      tdd_phase: red
    - id: AC-4
      title: measured single-entity history wall-time delta recorded in Validation
      status: open
      tdd_phase: red
---
## Goal

Make single-entity `aiwf history` / `aiwf show` fast by querying git *by path* with
changed-path bloom filters, and maintain those filters so the lever persists.

`aiwf history M-NNNN` currently greps every commit message (~1s over all 5,510
commits). `git log -- <path>` with changed-path bloom filters is ~14ms (measured on
this tree) — git skips the commits that never touched the file. The entity's path
set is *current path + prior paths* (archive `git mv`, `aiwf reallocate` renames),
which aiwf already tracks via `prior_ids` and the archive convention, so it can pass
the explicit path set rather than rely on `--follow` heuristics.

## Notes

- Reopens the M-0219 / G-0322 decision: that milestone measured only the *base*
  commit-graph (which git writes by default) and found ~1.5s; it never measured
  `--changed-paths` bloom filters + path-scoped queries, the orthogonal lever here.
- Maintenance: `git commit-graph write --reachable --changed-paths`, wired into
  `aiwf update` (or `git maintenance` config registered there), idempotent.
- Bloom filters are keyed by commit SHA (immutable) and shared across worktrees via
  the common object store — safe by construction; stale only ever means slower.
- Grep-fallback stays as the correctness oracle: path-scoped result must equal the
  trailer-grep result, including for renamed/archived entities.

### AC-1 — single-entity history resolves via path-scoped git log over the entity path set

### AC-2 — aiwf update maintains changed-path bloom filters idempotently

### AC-3 — path-scoped history equals trailer-grep history including renamed entities

### AC-4 — measured single-entity history wall-time delta recorded in Validation

