---
id: G-0687
title: acknowledge illegal --for-entity refuses a merge commit
status: open
discovered_in: M-0331
---
## What's missing

`aiwf acknowledge illegal <sha> --for-entity <id>` refuses a merge commit:
`internal/verb/acknowledgeillegal.go` verifies the binding with `git diff-tree`,
which names no file for a merge, so the verb answers that the SHA does not touch
the entity. Measured on a scratch repository where a merge collapses a side branch:
`aiwf acknowledge illegal <merge> --for-entity G-0001 --reason "..."` — expected an
acknowledgment bound to the entity, observed `SHA … does not touch entity G-0001
(its diff names no file resolving to G-0001; refusing operator-attested binding
without mechanical evidence)`, exit 2. The plain form without `--for-entity`
succeeds, and clears `entity-body-section-dropped` alone.

## Why it matters

`entity-body-section-dropped` can credit a merge, and its remedy says to add
`--for-entity <id>` when the same commit is also reported by
`provenance-untrailered-entity-commit`; a merge reported by both has no form of
the verb that clears both findings, so the pusher's only way past is
`--no-verify`, which is the boundary the hook exists to hold.
