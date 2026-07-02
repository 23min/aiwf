---
id: E-0054
title: 'Fast read paths: batch history walks and path-scope single-entity reads'
status: proposed
---
## Goal

Make aiwf's read verbs — `render`, `history`, `show` — fast in the devcontainer,
where they are subprocess-*wait* bound (the Docker/linuxkit `fork`/`exec` tax), by
cutting per-invocation git-subprocess count from O(entities × commits) to a single
shared history pass, and by letting git's own changed-path bloom filters skip
history for single-entity queries.

Measured on the kernel tree (5,510 commits, 657 pages, this devcontainer):
`aiwf render --format=html` takes **28 minutes** because it issues ~1,000+
per-entity `git log --grep` walks across **two** walk families — per-entity history
(`resolver.history`, N+2× per milestone) and per-milestone provenance/scopes
(`show.LoadEntityScopeViews`, 2 more full greps each). A throwaway single-pass spike
rendered **byte-identical** output in **12.8s** (~130×). Single-entity
`aiwf history` pays ~1s per full-history grep that a path-scoped `git log -- <path>`
with changed-path bloom filters reduces to ~14ms (measured).

This epic adds a derived *read strategy*, not a second source of truth. `git log` +
trailers stays canonical (per `design-decisions.md`); the design and per-lever
worktree/merge safety analysis live in
[`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md).

## Scope

In:

- **Render single unified history walk** — feed render's per-entity event lists
  *and* the authorize-opener map from one `BulkRevwalk`-shaped pass (both families,
  since batching history alone still timed out in the spike).
- **Path-scoped single-entity history + changed-path bloom-filter maintenance** —
  query `history`/`show` by the entity's path set (current + prior paths for
  archive/reallocate renames) and maintain bloom filters, reopening the
  M-0219 / G-0322 decision specifically for `--changed-paths` (that milestone
  measured only the base commit-graph and never the bloom-filter lever).

Out (deferred, tracked separately):

- Incremental `aiwf check` via a validated watermark (G-0323) — the persistent-cache
  tier; it must obey the SHA-keyed (never pointer-keyed) invariant in the perf doc
  and is a separate epic.
- Branch hygiene (G-0324) and allocator ref fan-out — cheap follow-ups.

## Constraints

- **Behavior-preserving.** Rendered output byte-identical to the per-entity path
  (proven technique; assert via full-tree diff / golden fixture).
- **No persistent cache in this epic.** Both milestones are pure recompute + git's
  own commit-graph, so they carry zero worktree/merge risk. Any future cache obeys
  the SHA-keyed invariant in the perf doc.
- No rule moves from pre-push to CI; no guarantee weakened.

## Source

- [`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md)
- E-0053 (prior perf epic; check-side subprocess collapse) and its deferred gaps
  G-0323 / G-0324 / G-0325.
- The render single-pass spike (28 min → 12.8s, byte-identical across 657 pages;
  2026-07-01).
