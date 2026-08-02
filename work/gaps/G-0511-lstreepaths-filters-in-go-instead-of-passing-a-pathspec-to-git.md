---
id: G-0511
title: LsTreePaths filters in Go instead of passing a pathspec to git
status: open
discovered_in: M-0286
---
## What's missing

`gitops.LsTreePaths` accepts `prefixes` but never passes them to git — it runs
`git ls-tree --full-tree -r --name-only -z <ref>` over the whole tree and
filters the result in Go. Every caller therefore pays for the entire tree
regardless of how narrow its interest is.

Two callers make that cost visible. `addCarriedUnder` asks for one move's
subtree; `dirtyPathsUnderMoves` now calls it for both ends of every candidate
move, so a sweep runs `2 × len(moves)` full-tree listings where it previously
ran `len(moves)`. `recordedEntityPaths` adds one more.

Measured on a 2478-file tree with three sweep candidates: a full-tree listing
is ~52ms against ~38ms for a pathspec-scoped one, and `aiwf archive`
end-to-end grew ~230ms. The shape is `(moves + 1) × full-tree`, so a
50-candidate sweep would spend seconds in listing alone.

## Why it matters

The cost is linear in tree size *and* in candidate count, and the second
factor is the one that grew. A first-run sweep against a large pre-archive
tree is exactly the case with the most candidates and the biggest tree, so
the two factors multiply where it hurts most.

Nothing is incorrect today — this is throughput, not behaviour. It is filed
rather than fixed inline because `LsTreePaths` is shared with callers outside
the sweep, and narrowing what git returns changes what every one of them
sees; that needs its own verification rather than riding along in a
milestone about decline correctness.

## Sketch

Pass `prefixes` to git as a pathspec (`git ls-tree … <ref> -- <prefix>…`)
instead of filtering in Go, keeping the no-prefix call a full-tree listing.
Callers that pass no prefix are unaffected. The evidence is a call-count or
timing assertion, since a correctness test cannot distinguish the two.

Note the one caller that must keep asking for everything:
`recordedEntityPaths` deliberately offers every recorded path to
`entity.PathKind` rather than pre-filtering by directory, so the record's
view of what counts as an entity file cannot drift from the loader's.
