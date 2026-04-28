---
name: aiwf-history
description: Use when the user asks "what happened to <entity>" or wants the timeline of an aiwf entity. Runs `aiwf history` which reads `git log` filtered by structured commit trailers.
---

# aiwf-history

The `aiwf history` verb answers "what happened to this entity?" by filtering `git log` for the entity's commit trailers. There is no separate event log; the git log is the time machine, made queryable by `aiwf-verb` / `aiwf-entity` / `aiwf-prior-entity` trailers.

## When to use

The user wants the lifecycle of one entity. Example phrasings: "when was M-007 created?", "show me what happened to E-19", "why is this gap closed?".

## What to run

```bash
aiwf history <id>            # one line per event
aiwf history <id> --format=json
```

The output is one event per line: `DATE  ACTOR  VERB  DETAIL  COMMIT`. DETAIL is shaped per verb — the title for `add`, `old → new` for `promote`, `→ cancelled` for `cancel`, `slug → <new>` for `rename`, `<old-id> → <new-id>` for `reallocate`.

## What aiwf does

1. Runs `git log` with a grep for `aiwf-entity: <id>` OR `aiwf-prior-entity: <id>` trailers (so reallocate events surface from both the old and new id's history).
2. Parses each matching commit's subject + trailers + author date.
3. Renders one event per line.

## Limitations

- `aiwf history` shows only verb-driven events. Hand-edits to the markdown file won't have trailers and will not appear. To see byte-level history of a file, use `git log -- <path>`.
- After a reallocate, query both ids if you want the full picture; the new id's history starts from the reallocate event.

## Don't

- Don't try to reconstruct history from filesystem timestamps — `git log` is authoritative.
- Don't expect prose-body changes to show up. Only frontmatter mutations through aiwf verbs are queryable here.
