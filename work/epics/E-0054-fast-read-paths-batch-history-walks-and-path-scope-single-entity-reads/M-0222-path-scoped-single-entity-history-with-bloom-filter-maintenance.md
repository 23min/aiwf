---
id: M-0222
title: Path-scoped single-entity history with bloom-filter maintenance
status: draft
parent: E-0054
tdd: required
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
