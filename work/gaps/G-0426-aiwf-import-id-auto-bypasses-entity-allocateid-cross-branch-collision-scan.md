---
id: G-0426
title: aiwf import id auto bypasses entity.AllocateID cross-branch collision scan
status: open
priority: high
---
## What's missing

`internal/verb/import.go` hand-rolls id allocation (`idPrefix`, `formatID`, `parseIDInt`, `computeHighestPerKind`) instead of calling `entity.AllocateID`, and never consults `tree.Tree.AllocationIDs()`. Trunk collisions are still caught (`Import` runs `projectionFindings`, whose `idsUnique` reads `TrunkIDs`), but `id: auto` — the documented first-class idiom for greenfield entities — never sees ids allocated on sibling local branches or pushed-but-unmerged remote branches. `importcmd` also lacks the `--fetch` flag `add` has. The migration doc overclaims that auto allocation "never collides by construction."

## Why it matters

This reintroduces, for `import` only, exactly the pre-`AllocateID` cross-branch collision exposure the id-lifecycle work closed for `add`: a collision minted via import surfaces only later, at merge time, as an `ids-unique` finding needing `aiwf reallocate`. Finding F8 of `docs/initiatives/verb-layer-cleanup.md` (adversarially verified); the fix is routing import's auto-id path through `entity.AllocateID`, inheriting its existing collision-avoidance tests.
