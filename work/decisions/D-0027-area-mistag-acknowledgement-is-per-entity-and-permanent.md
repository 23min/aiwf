---
id: D-0027
title: Area-mistag acknowledgement is per-entity and permanent
status: proposed
---
## Context

M-0181 adds `aiwf acknowledge mistag <id>`, which suppresses the `area-mistag`
warning for an entity. The suppression is keyed purely on the entity id and is
permanent: once acknowledged, the entity never fires `area-mistag` again,
regardless of which areas its future commits land in or whether its `area` tag
later changes.

The independent `wf-rethink` of the acknowledge-namespace flagged that this
diverges from the illegal-ack family's own evolution: G-0231 added a scoped
per-(SHA, entity) shape precisely to bound an acknowledgement to what it
blessed. A scoped mistag ack — key (entity, blessed-foreign-area), suppressing
only while the foreign set still matches — would be self-healing: a later,
genuinely-different mistag on the same entity would re-fire.

## Decision

Keep the per-entity, permanent shape for M-0181.

- `area-mistag` is a non-escalating **warning** (deliberately absent from
  `ApplyAreaRequiredStrict`), so a stale suppression costs a missed advisory,
  never a blocked push.
- The common "the tag was simply wrong" case is already handled by re-tagging
  via `aiwf set-area`, after which the mistag no longer fires and the ack is
  moot.
- Per-entity matches the operator's act ("this entity does cross-cutting work")
  and mirrors `acknowledge-illegal`'s per-SHA permanence.

## Known alternative (the answer if friction appears)

If stale-suppression friction is ever observed in practice, the scoped
(entity, blessed-foreign-area) shape is the known refinement — one extra
trailer plus a foreign-set comparison in `WalkAcknowledgedMistags`, consistent
with G-0231's per-(SHA, entity) precedent. Deliberately not built now (YAGNI
until the friction is real).
