---
id: G-0513
title: Masked-terminal report misses a candidate unparseable on disk
status: open
discovered_in: M-0286
---
## What's missing

A candidate that is terminal in the record and unparseable on disk is neither
swept nor reported. The verb answers:

```
aiwf archive: no terminal-status entities awaiting sweep (tree is converged)
```

with a sweep genuinely due against the record.

The masked-terminal report walks the loaded tree, so it cannot see an entity
the loader dropped — which is precisely the class the referrer scan was
widened to the record to catch. One consumer of the divergence set now reads
the record as well as the working tree; this one still reads only the working
tree.

## Why it matters

"Tree is converged" is the one answer that is false here, and it is the same
message the masked-terminal report was built to eliminate for the
edited-status route. That route is covered; the unparseable route is not.

Nothing is written either way, and `aiwf check` does report a load error for
the file, so the operator is not blind — this is a reporting gap rather than
damage. It is worth closing because the two halves of one decision now
disagree about which entities exist, which is the coherence problem this
epic's last milestone was about.

## Sketch

Give the masked-terminal report the same candidate set the referrer scan
uses: the loaded tree's entities plus the entity paths the record carries.
For a path absent from the loaded tree there is no working-copy status to
compare, so the report says the record holds a terminal status the working
copy cannot confirm, rather than asserting a status disagreement it cannot
compute.

The evidence is a repo with a terminal entity at HEAD whose working copy does
not parse, asserting the verb names it rather than reporting convergence.
