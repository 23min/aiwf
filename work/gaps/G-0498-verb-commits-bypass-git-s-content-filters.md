---
id: G-0498
title: Verb commits bypass git's content filters
status: open
priority: low
discovered_in: M-0284
---
## What's missing

The verb commit path stores the working copy's bytes verbatim. `gitops.CommitVerbChange` hands
content to `CommitTree`, which writes blobs directly, so git's clean filter never runs — unlike
`git add`, which applies it.

In a repo carrying content filters the two conventions disagree permanently. Measured with
`core.autocrlf=true` (the Git-for-Windows installer default): a tree seeded by `git commit`
holds LF blobs, checkout smudges the working copy to CRLF, and `git status` reports the tree
clean — while any aiwf verb would store the CRLF bytes, rewriting every line ending in the file
it touches under its own trailer.

## Why it matters

Two consequences, and the second is the one that bites.

A verb silently rewrites content it was not asked to change. The diff is invisible to `git
status` beforehand and shows as a whole-file change afterwards, attributed to whatever verb ran.

And the commit-side guard, correctly, refuses it. The guard predicts what the commit would
carry, the commit would carry different bytes than the record holds, so it reports divergence
on a tree the operator has not touched. The refusal is accurate and the remedies it offers do
nothing, because there is no uncommitted edit to commit or discard. On such a repo every
mutating verb is unusable.

The guard is not the defect — it is the surface that makes this visible. Comparing filtered
bytes instead would hide it again while leaving the silent rewrite in place.

## Scope

The question is which convention aiwf adopts for the content it stores.

- Apply the clean filter when building blobs, matching `git add`, and compare filtered on both
  sides. Round-trips correctly with every other git tool.
- Declare the tree filter-exempt via a shipped `.gitattributes` (`-text`) for entity paths, so
  no filter applies to the files aiwf owns and the raw-byte comparison is right by construction.

Out of scope: the comparison primitive's own shape, which is settled — it matches whatever the
commit path stores, and follows that decision rather than leading it.

## References

- `internal/gitops/committree.go` — where blobs are built
- `internal/gitops/divergence.go` — the comparison that surfaces the mismatch
- ADR-0038 — the guard whose accuracy depends on this answer