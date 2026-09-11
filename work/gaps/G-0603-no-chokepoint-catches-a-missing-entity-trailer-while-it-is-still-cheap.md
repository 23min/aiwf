---
id: G-0603
title: No chokepoint catches a missing entity trailer while it is still cheap
status: addressed
discovered_in: M-0312
addressed_by_commit:
    - fc5501d
---
## What's missing

The skill-edit provenance backstop is commit-scoped by design — provenance is a property a
commit carries, so an uncommitted edit has none. The consequence is that a forgotten
`aiwf-entity` trailer is discovered only at the next gate run, by which point the commit
exists and the repair is an amend or a rebase. On a branch the base resolves to the
merge-base with trunk, so the fault persists for the branch's whole life until history is
rewritten.

Nothing catches the omission at the moment it is cheap to fix.

## Why it matters

The repo's own enforcement doctrine is to fire as early as the class allows. This class
can fire at commit composition: `aiwf init` already materializes a `commit-msg` hook, and
`git diff --cached --name-only` tells that hook whether the staged change touches a
watched surface. A one-second refusal at composition replaces an N-commit rebase.

The violation Detail currently says only "re-commit", which under-states the repair once
more commits sit on top.

## Options

Extend the materialized `commit-msg` hook to refuse a staged watched-surface edit whose
message carries no `aiwf-entity` trailer. The hook already parses trailers for the
verb-value check, so the parsing is in place.

Two things to settle: the hook runs in consumer repos, where this rule is meaningless, so
it needs the same aiwf-repo-only scoping the policy has; and a commit-msg hook cannot see
whether the named id resolves without loading the tree, so it would enforce presence while
the CI-tier policy continues to enforce resolution.
