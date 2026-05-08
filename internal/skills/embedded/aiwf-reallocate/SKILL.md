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

When passing an id and two entities share it, `aiwf` runs the trunk-ancestry tiebreaker:
- If exactly one side's add commit is an ancestor of the configured trunk ref, that side keeps the id (the team has been calling it that name) and the OTHER side is renumbered automatically.
- If both/neither are on trunk, `aiwf` refuses with an "ambiguous" error listing both candidate paths and the diagnostic. Pass a path to disambiguate.

Sandbox repos with no trunk in scope skip the tiebreaker — operators always pass a path there.

The id format never gets a suffix (no `M-007a`); collision recovery always picks `max + 1` against the working tree ∪ trunk.

## What aiwf does

1. Picks the next free id for the kind (`max + 1` at call time, scanning the working tree ∪ the configured trunk ref so the new id can't collide with trunk either).
2. `git mv` the colliding file or directory to its new path.
3. Rewrites every reference field in every other entity's frontmatter (e.g., a milestone's `parent`, a gap's `addressed_by`).
4. Appends the OLD id to the renumbered entity's `prior_ids:` frontmatter list (oldest-first). The list is the tree-level source of truth for lineage; tree-only readers (`aiwf show`, the HTML render, future projections) read it directly without shelling out to git log.
5. Reports body-prose mentions of the old id as findings — those need a human eye, since prose is not parsed.
6. Commits with trailers `aiwf-verb: reallocate`, `aiwf-entity: <new-id>`, `aiwf-prior-entity: <old-id>`, `aiwf-actor: <actor>`.

`aiwf history <old-id>` still works after a renumber: the cmd dispatcher resolves the queried id through `prior_ids`, walks the full chain (every prior id plus the current id), and runs one `git log` grep against the union. Both the original id and the current id return the same chronological timeline — pre-rename commits, the rename commit, and post-rename commits, in order.

## Don't

- Don't try to fix a collision by hand-editing the id in frontmatter. References in other files won't follow.
- Don't reallocate "to clean up" non-collisions. Only call this verb when an `ids-unique` finding exists; renumbering otherwise rewrites history for no benefit.
