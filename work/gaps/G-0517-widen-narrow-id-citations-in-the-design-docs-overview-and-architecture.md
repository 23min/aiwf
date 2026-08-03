---
id: G-0517
title: Widen narrow id citations in the design docs, overview, and architecture
status: open
---
## Problem

Three active documentation paths still carry below-canonical-width ids, left
unswept when the doc corpus was linted: `docs/design/**`,
`docs/overview.md`, and `docs/architecture.md`.

They are excluded for a reason rather than by oversight. The swept corpus was
tutorial fiction, where the fix is the canonical placeholder. These are mostly
citations of entities that were genuinely real at a narrow width, so the
correct fix is widening each to the real canonical id — a different edit,
made one reference at a time, with a lower payoff than cleaning the docs an
assistant reads to learn the workflow.

## Why it is not simply added to the lint

The `doc-id-width` corpus is configured in `aiwf.yaml`, so adding these paths
is a one-line change. What that would surface is a worklist of citations
needing individual research: for each, whether the narrow id names an entity
that still exists at canonical width, one archived at its original narrow
width (read tolerance is permanent, so those are correct as written), or
nothing at all.

## Resolution

Widen each citation to the entity's canonical id, leaving genuinely-narrow
archived references alone, then add the three paths to `docs.paths` so the
lint holds them.
