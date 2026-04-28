---
name: aiwf-reallocate
description: Use when `aiwf check` reports an `ids-unique` finding (two entities share an id, typically after a branch merge). Runs `aiwf reallocate` to renumber one of them and rewrite all references.
---

# aiwf-reallocate

The `aiwf reallocate` verb resolves id collisions by picking the next free id, `git mv`-ing the colliding entity, and walking every other entity's frontmatter to rewrite reference fields. Body-prose mentions are surfaced as findings — humans review those.

## When to use

`aiwf check` reported `ids-unique` (or the pre-push hook blocked a push for the same reason). Two files claim the same id — almost always because two branches independently allocated the id and were then merged.

## What to run

```bash
aiwf reallocate <id>
# or, when the id is ambiguous (both entities share it):
aiwf reallocate <path>
```

When passing an id, `aiwf` errors if it cannot pick exactly one entity to renumber. Pass the path of the loser to disambiguate. The id format never gets a suffix (no `M-007a`); collision recovery always picks `max + 1`.

## What aiwf does

1. Picks the next free id for the kind (`max + 1` at call time).
2. `git mv` the colliding file or directory to its new path.
3. Rewrites every reference field in every other entity's frontmatter (e.g., a milestone's `parent`, a gap's `addressed_by`).
4. Reports body-prose mentions of the old id as findings — those need a human eye, since prose is not parsed.
5. Commits with trailers `aiwf-verb: reallocate`, `aiwf-entity: <new-id>`, `aiwf-prior-entity: <old-id>`, `aiwf-actor: <actor>`.

The `aiwf-prior-entity` trailer is what lets `aiwf history <old-id>` still find this event.

## Don't

- Don't try to fix a collision by hand-editing the id in frontmatter. References in other files won't follow.
- Don't reallocate "to clean up" non-collisions. Only call this verb when an `ids-unique` finding exists; renumbering otherwise rewrites history for no benefit.
