# Epic wrap — E-0054

**Date:** 2026-07-03
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0054-fast-read-paths-single-pass-render-walk-and-read-verb-grep-guard
**Merge commit:** b565acc2

## Milestones delivered

- M-0223 — Guard the unconditional authorize-opener grep in the read verbs (merged 7937fd05)
- M-0221 — Single unified history walk for render (merged abea4dc7)

M-0222 — Path-scoped single-entity history with bloom-filter maintenance — was **cancelled
pre-start**; measurement showed the path-scoping lever needs the query-equivalence gaps
(pathless commits, path-set derivation, history simplification) handled first, so its scope
was deferred whole to G-0340 rather than started.

## Summary

E-0054 attacked read-path latency, which scaled with total git history because every read
re-derives from `git log`. Two levers landed, both byte-identical to the prior behaviour:
M-0223 guards the repo-wide authorize-opener grep in `aiwf history` / `aiwf show` so it runs
only when the entity's loaded events carry scope data (~44% / ~32% off a scopeless entity),
and consolidates the two near-duplicate grep impls into one shared `cliutil.AuthorizeOpeners`.
M-0221 collapses `aiwf render`'s per-entity history fan-out (~N-per-milestone `git log`
walks) into ONE shared HEAD pass (`check.WalkHeadCommits` + `%aI`/`%s`) bucketed in memory —
**~35 min → ~4.5s** on the 688-page kernel tree (~466×). The single-pass reuses M-0223's
consolidated primitives (no fourth copy). The persistent-cache tier that would flatten reads
regardless of size was deliberately left deferred (G-0323) — it carries the worktree/merge
safety risk the batching levers avoid.

## ADRs ratified

- none — the performance doctrine (batching vs the deferred SHA-keyed cache, and the
  "cache only immutable SHA-keyed facts" invariant) lives in
  [`docs/pocv3/design/performance.md`](../../../docs/pocv3/design/performance.md); each
  milestone's design calls (fail-loud render error semantic, the `check.HeadCommit` reuse,
  the shared-primitive seams — `wf-rethink`-reviewed) are in its spec's Reviewer notes.

## Decisions captured

- none — see "ADRs ratified".

## Follow-ups carried forward

- G-0340 — Path-scoped single-entity history acceleration with bloom-filter maintenance
  (the cancelled M-0222 scope; the highest-value remaining single-entity read lever, gated
  on the query-equivalence work).
- G-0323 — SHA-keyed read-model cache + per-SHA incremental check (the "flat regardless of
  size" tier; deferred, higher effort, safe only under the caching invariant).
- G-0324 — branch hygiene (prune merged local branches; shrinks allocator + check fan-out).

## Handoff

The two risk-free batching levers are shipped and `performance.md`'s "Recommended sequence"
is updated to mark them so. The next read-path work is G-0340 (path-scoping) then G-0323
(persistent cache); do 0–1 first and re-measure before investing in the cache tier (YAGNI).
Nothing in E-0054 is left half-done — M-0222's scope moved whole to G-0340.
