---
id: G-0499
title: Archive's referrer scan misses referrers absent from the loaded tree
status: addressed
discovered_in: M-0284
addressed_by_commit:
    - f8b80cf8a
---
## What's missing

`dirtyEntityPaths` builds its candidate list from `tr.Entities` — the working tree's loaded
entities. A referrer present at HEAD but absent from that list is never queried, never lands in
the dirty set, and never reaches `moveBlockers`' referrer loop. `planArchiveRewrites` iterates
the same set, so no link rewrite is emitted either, and the sweep commits the move alone.

Measured, three ways into the same end state — the referrer deleted on disk, hand-renamed, or
carrying momentarily unparseable frontmatter:

    aiwf archive --apply   ->  sweep 1 entity into archive/   exit 0, no Skipped: line
    HEAD:  work/gaps/G-0002-linking-gap.md  links to  work/gaps/G-0001-target-gap.md
           work/gaps/archive/G-0001-target-gap.md          <- target absent at the linked path

The unparseable-frontmatter route is the ordinary one: mid-edit YAML is briefly invalid as a
matter of routine, and editing an entity body in the working tree before blessing it is the
rhythm the shipped guidance recommends.

## Why it matters

The damage is permanent. Once the target is archived, `IsArchivedPath` excludes it from every
later scan, so a re-run reports the tree converged and never repairs the link. `aiwf check`
reports zero errors on the result. Restoring the referrer changes nothing.

This is the exact class the per-candidate decline was built to close, and the milestone that
built it records the class as closed. The bodies are compared against HEAD, as recorded — but
the candidate list they are drawn from is still working-copy-derived, so the comparison never
runs for the referrers that matter.

## Scope

Enumerate referrer candidates from the record as well as from the working tree, the same
correction already applied to the carried half of a move. `gitops.DivergentPaths` already
reports `DivergenceAbsentFromDisk` for this shape; it is unreachable through the current
candidate list.

Three neighbouring defects share the root cause — `moveBlockers`' predicate not matching
`planArchiveRewrites`' — and belong with it rather than separately:

- A working-copy-only link (HEAD has none) leaves the move in the plan, so the commit-side
  guard refuses the whole verb and names no candidate; `--dry-run` promises a sweep `--apply`
  refuses.
- An archived referrer mid-edit blocks an unrelated sweep, because `moveBlockers` lacks the
  `IsArchivedPath` filter `planArchiveRewrites` has.
- The sweep enumerates only a move's source, never its destination, so an untracked file
  sitting at the destination is invisible to the decline and refused wholesale at the commit.

Out of scope: the comparison primitive and the claim-side guard, both settled.

## References

- `internal/verb/archive.go` — `dirtyEntityPaths`, `moveBlockers`, `planArchiveRewrites`
- ADR-0038 — the per-candidate scoping this is meant to implement
- M-0284 — where the class was recorded as closed