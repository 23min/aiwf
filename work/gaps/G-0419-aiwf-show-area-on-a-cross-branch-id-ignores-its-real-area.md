---
id: G-0419
title: aiwf show --area on a cross-branch id ignores its real area
status: open
discovered_in: M-0260
---
## What's missing

`aiwf show <id> --area <A>` resolves the entity's --area predicate via
tr.ResolvedAreaByID(id), which only consults the local tree. For an id
resolved cross-branch (M-0260), this always returns "" (untagged),
regardless of the entity's real `area:` field on the ref it actually
resolved from — so `aiwf show <cross-branch-id> --area X` always
reports the entity as untagged/out-of-area, even when its real content
would place it in X.

## Why it matters

This is a narrow, unlikely-to-be-hit flag combination (an operator
targeting a specific area filter against an id they already know is
cross-branch-known), but it's a real correctness gap: the --area
predicate silently gives a wrong answer instead of erroring or
resolving correctly. Fixing it needs the resolved cross-branch
entity's own Area field (already read in buildCrossBranchShowView, via
entity.Parse) to be threaded into the --area predicate instead of
tr.ResolvedAreaByID's local-only lookup. Discovered during M-0260's
independent design review; deferred as out of that milestone's stated
scope.