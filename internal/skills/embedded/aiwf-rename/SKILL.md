---
name: aiwf-rename
description: Use when the user wants to rename an entity's slug (the human-readable suffix in the file/dir name). Runs `aiwf rename` so the id is preserved and references stay valid.
---

# aiwf-rename

The `aiwf rename` verb changes the slug portion of an entity's path while preserving the id. References to the entity (which use the id) keep working.

> **Looking to change a title?** For changing an entity's title (the prose label, distinct from the slug), use `aiwf retitle <id> <new-title>` — that is the dedicated verb for title mutations. This skill covers slug renames only. (The two verbs stay separate by design — single-mutation rule keeps reasoning local.)

## When to use

The user wants the file or directory name to read better but the entity itself isn't changing identity. Examples: a milestone was named `M-003-things` and they want it to be `M-003-acceptance-criteria`.

## What to run

```bash
aiwf rename <id> <new-slug>
```

`aiwf` normalizes the new slug into kebab-case (lowercases, ASCII-only, runs of non-alphanumerics collapse into single hyphens, trailing hyphens trimmed). `"Acceptance Criteria!"` becomes `acceptance-criteria`. The verb refuses only when normalization yields the empty string or the same slug as the current path.

## What aiwf does

1. Looks up the entity by id, computes the new path (`<dir>/<id>-<new-slug>.md` or `<dir>/<id>-<new-slug>/`).
2. `git mv` the file or directory to the new path.
3. Commits with trailers `aiwf-verb: rename`, `aiwf-entity: <id>`, `aiwf-actor: <actor>`.

The frontmatter `id` does not change. The frontmatter `title` does not change either; if the user wants the displayed title updated too, edit it separately.

## Don't

- Don't `git mv` by hand — you'll skip the trailer and `aiwf history` won't show the rename.
- Don't try to rename across kinds (e.g., turn a gap into an epic). That's a different operation; create a new entity and link via `addressed_by` or `relates_to`.
- Don't change the id portion of the slug. Use `aiwf reallocate` for that.
