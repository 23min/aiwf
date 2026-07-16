---
id: G-0418
title: Cross-branch hit/collision scan is duplicated across 3 call sites
status: open
discovered_in: M-0260
---
## What's missing

The "union local+remote ref hits, then run DetectCollisions over that
same union" composition recipe is copied at three independent call
sites: internal/cli/cliutil/treeload.go (eager, aiwf check's path),
internal/cli/show/show.go's buildCrossBranchShowView, and
internal/cli/list/list.go's crossBranchListRows. The underlying
primitives (trunk.LocalRefHits, RemoteRefHits, DetectCollisions,
DistinctRefs) each live once in internal/trunk — only the composition
of "the hits passed to DetectCollisions must be exactly the union that
was scanned" is triplicated.

## Why it matters

The coupling between the hits union and the collision-detection input
is load-bearing: a future change that adds a third ref source (e.g.
stash refs) to one site's union but not its DetectCollisions call
would silently under-detect collisions at that one site while the
other two stay correct. A single trunk-level helper (e.g.
ScanCrossBranch(ctx, root) (hits []RefHit, collisions map[string]bool))
would collapse the recipe to one place without imposing eager cost on
show/list's lazy callers, making the coupling atomic. Discovered during
M-0260's independent design review; not urgent (the duplication is ~2
lines per site and drift is slow), but the right eventual shape.