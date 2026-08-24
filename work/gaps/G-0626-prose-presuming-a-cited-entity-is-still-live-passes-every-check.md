---
id: G-0626
title: Prose presuming a cited entity is still live passes every check
status: open
priority: medium
---
## What's missing

A body citation is resolved to the cited entity and its status is never read.
`internal/check/body_prose_id.go` builds `BodyProseIDIndex`, a canonical-id map to
`*entity.Entity` covering the whole tree, then judges the token's *shape* — whether
it names an entity at all — and stops. The resolved entity carries `Status`; the
rule's only `Status` occurrence is `ast.WalkStatus`. So a body may state that work
waits on an entity that is already terminal, and the tree reports clean.

```
grep -rnoE '(until|once|blocked on|pending|awaiting) (G|M|E|D|ADR)-[0-9]{4}' work/gaps/*.md
```

Run 2026-08-24 this returns four sentences. Three cite an entity that has since
gone terminal, and each was true at the moment it was written:

| body | cites | target status | sentence written | target went terminal |
|---|---|---|---|---|
| G-0215 | `until M-0159` | `done` | 2026-06-02 20:56 | 2026-06-02 23:30 |
| G-0420 | `until G-0078` | `addressed` | 2026-07-16 | 2026-07-17 |
| G-0472 | `until G-0557` | `addressed` | 2026-08-05 | 2026-08-06 |

Widening the phrasing to the rest of the not-yet vocabulary (`after`, `before`,
`depends on`, `gated on`, `remains open`, `is proposed`, `will land`) and resolving
each target's status raises it to 18 sentences across 14 open gap bodies, measured
the same day. The sharpest is G-0073, which sequences an implementation epic on
"Once ADR-0003 is ratified" — ADR-0003 is `rejected`, and ADR-0045 records the
decision against a seventh entity kind.

`aiwf check` reports no finding for any of them.

## Why it matters

A gap body is read as current truth and planned against. A sentence deferring work
until a cited entity lands sends its reader to wait for something already finished,
and the deferral is invisible in both directions: promoting an entity to terminal
reaches none of the bodies that presume it live, and reading one of those bodies
gives no signal that its premise has expired.

The window is too short for a review cadence to cover. The three instances above
expired 2 hours 34 minutes, one day, and one day after they were written. This is
not records aging out — it is the tree's own closure rate falsifying prose faster
than anyone re-reads it, so every promote is a candidate falsifier for every
deferral clause in the backlog.
