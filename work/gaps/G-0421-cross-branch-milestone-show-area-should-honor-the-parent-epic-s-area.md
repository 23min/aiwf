---
id: G-0421
title: Cross-branch milestone show --area should honor the parent epic's area
status: open
priority: low
discovered_in: M-0266
---
## What's missing

M-0266 fixed `aiwf show <id> --area <A>` for cross-branch ids whose kind
carries its own `area:` (epic, gap, ADR, decision, contract): the predicate
now evaluates against the resolving ref's real area. A cross-branch
**milestone** is still reported untagged, because a milestone never stores its
own `area:` — its area rolls up from the parent epic (tree.ResolvedArea), and
buildCrossBranchShowView reads the entity's own parsed `resolved.Area` (empty
for a milestone) rather than following that roll-up across refs. So `aiwf show
<cross-branch-milestone> --area X` reports untagged even when the parent
epic's area is X.

## Why it matters

Narrow but real: an operator filtering cross-branch milestones by area sees
them as untagged — an asymmetry with the local-milestone path, which rolls up
to the parent epic via tr.ResolvedAreaByID. It is not a regression (before
M-0266 every cross-branch id reported untagged) and was out of M-0266's stated
scope, whose design note deliberately threads the entity's own `resolved.Area`
and whose AC evidence is a gap fixture. Fixing it needs the cross-branch read
to resolve a milestone's area from its parent epic: straightforward when the
parent epic is local, but the parent may itself be cross-branch, so the fix
must decide how far to follow the parent chain across refs. Discovered during
M-0266's independent review.
