---
id: G-0434
title: resolveViaPriorIDs prefers a stale prior_ids match over a reused live id
status: open
priority: medium
discovered_in: M-0126
---
## Problem

`internal/check/provenance.go`'s `resolveViaPriorIDs(id, t)` (used by `RunPromoteOnWrongBranch` via `resolvedID := resolveViaPriorIDs(walkRenameChain(entityID, renameChain), t)`) resolves a commit trailer's entity id by unconditionally preferring a `t.ByPriorID(id)` match over the id's *live*, currently-occupied entity. This is backwards precedence once an id is freed by `aiwf reallocate` and later reused for an unrelated new entity: the freed id's *old* occupant still carries `prior_ids: [<freed-id>]` in its current frontmatter, so `t.ByPriorID(<freed-id>)` keeps matching the old entity forever, even after the id has been legitimately reassigned to something else.

Concretely reproduced in this repo: id `M-0126` was originally allocated to a milestone under E-0033 ("Implement fsm-history-consistent check rule..."), later reallocated to `M-0130` during an id-collision resolution (commit `d439f97`, 2026-05-19). `M-0126` was then reused for an unrelated new milestone under E-0034 ("Triage docs/pocv3/..."). Every commit trailered `aiwf-entity: M-0126` after that reuse — including a completely correct `aiwf promote M-0126 draft -> in_progress` on the E-0034 epic branch — gets `resolvedID` incorrectly resolved to `M-0130` (because `M-0130.prior_ids` still lists `M-0126`), which then computes the *wrong* expected parent branch (`epic/E-0033-...` instead of `epic/E-0034-...`) and fires a false-positive `promote-on-wrong-branch` warning.

`internal/tree` already has the correct precedence for exactly this situation: `Tree.ResolveByCurrentOrPriorID` tries `ByID` (current/live id) first, falling back to `ByPriorID` (lineage match) only on a miss, with a doc comment describing precisely the id-reuse case. `resolveViaPriorIDs` in `internal/check/provenance.go` reimplements a narrower version of the same idea but gets the precedence backwards, unconditionally preferring the prior-ids match.

## Direction

Fix `resolveViaPriorIDs` (`internal/check/provenance.go:368`) to check `t.ByID(id) != nil` first and return `id` unchanged in that case, only falling through to the `t.ByPriorID` lookup when the direct lookup misses — mirroring (or delegating to) `Tree.ResolveByCurrentOrPriorID`'s precedence. Add a regression fixture: two entities sharing a numeric id across time (one reallocated away, the id reused by a new entity), asserting a promote commit on the *new* entity's own id resolves to itself, not the old reallocated-away entity.