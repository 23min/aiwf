---
id: G-0628
title: A closure invalidates claims in citing records and nothing tells either side
status: addressed
priority: high
addressed_by_commit:
    - 5f5d9cb20
---
## What's missing

`aiwf promote <id> <terminal>` computes which records cite the entity it is closing,
and discards the result. The path is `projectionFindings` → `check.Run` →
`bodyProseID`, which reads every entity body and resolves each id token against
`BodyProseIDIndex`. Nothing in the verb, in `internal/check/`, or in the rituals'
closure steps compares a citation against the cited entity's state.

So a closure lands and the records that cite it are not told, in either direction.
The closer holds context about the entity being closed and not about the ones citing
it; whoever later opens a citing record holds the reverse, and by then what the
closure implied is gone.

One instance, reproducible:

```
git log --all --format='%ad %s' --date=short --grep='aiwf-entity: G-0073' | grep edit-body | head -1
git log --all --format='%ad %s' --date=short --grep='aiwf-entity: ADR-0003' | grep -i reject
```

G-0073's body was last written 2026-08-12 and sequences an implementation epic on
"Once ADR-0003 is ratified". ADR-0003 was promoted to `rejected` on 2026-08-16, and
ADR-0045 records the decision against a seventh entity kind. `aiwf check` reports the
tree clean.

Measured 2026-08-24 by comparing each open gap's last `add`/`edit-body` commit against
the terminal-promote date of every entity its body cites: **36 of the 161 open gaps
with a known body-write date cite an entity that went terminal afterwards**. Among
them G-0212 → E-0030, G-0111 → M-0130, G-0121 → G-0567.

The reference graph cannot see any of this. `entity.ForwardRefs` reads frontmatter
only, and across 619 gap files exactly two gap-to-gap frontmatter edges exist against
786 prose citations from the open set alone.

## Why it matters

A gap body is read as current truth and planned against, so a stale premise produces
confident wrong work while the tree reports clean. The cost is already in the record:
one gap sequences work on an ADR that was rejected, and three more rest on milestones
and gaps that closed after the sentence was written.

The rate is structural rather than occasional. 224 gaps closed in the two months to
2026-08-24, against a backlog that has not shrunk, so every closure is a candidate
falsifier for every open body that cites it — and the two moments when someone could
notice are the moments nobody is looking.
