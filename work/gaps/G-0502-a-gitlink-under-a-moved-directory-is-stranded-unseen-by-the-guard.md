---
id: G-0502
title: A gitlink under a moved directory is stranded, unseen by the guard
status: open
discovered_in: M-0284
---
## What's missing

`DivergentPaths` keeps only `blob` entries when reading what HEAD records, so a
gitlink (mode `160000`) is never recorded and never yields `DivergenceAbsentFromDisk`.
`LsTreePaths` does put the gitlink into the carried set; it simply never becomes a
divergence.

That is exactly the property the HEAD side of `planCarriedPaths` was added to
prevent, and its own comment claims: a path the record carries and the working tree
lacks is one the commit strands at its old location while its siblings move.

Measured, a gitlink recorded under a contract directory that is then archived:

    before:  100644 blob   … work/contracts/C-0001-probe/contract.md
             160000 commit … work/contracts/C-0001-probe/vendor
    aiwf archive --apply   ->  exit 0
    after:   160000 commit … work/contracts/C-0001-probe/vendor          <- stranded
             100644 blob   … work/contracts/archive/C-0001-probe/contract.md

`aiwf check` reports 0 errors on the split tree.

A gitlink whose directory *is* present on disk takes the other arm and dies with
`comparing …/vendor: is a directory, not a file` — safe, but with no remedy and no
hint that a submodule is the cause.

## Why it matters

Low reachability: it needs a submodule under `work/`. What makes it worth tracking
is that it falsifies a claim the guard's own doc comment makes, so a reader auditing
that comment would conclude the class is closed when one member of it is not.

The second arm is a usability problem rather than a correctness one — an operator
meeting it has no way to learn what happened.

## Scope

Decide what a non-blob HEAD entry means to the comparison. A gitlink is a real thing
the commit would strand, so reporting it as absent-from-disk is defensible; so is
declining the candidate move outright, since the commit path cannot record a gitlink
any more than it can record a symlink. The directory-shaped arm should say what it
found either way.

Out of scope: submodule support as a feature. This is about the guard telling the
truth, not about aiwf managing submodules.

## References

- `internal/gitops/divergence.go` — the blob-only filter and the directory arm
- `internal/verb/apply.go` — `planCarriedPaths`, whose comment claims this class
- G-0499 — the archive sweep's other record-versus-working-tree gaps