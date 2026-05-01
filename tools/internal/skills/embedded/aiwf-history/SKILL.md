---
name: aiwf-history
description: Use when the user asks "what happened to <entity>" or wants the timeline of an aiwf entity (or acceptance criterion). Runs `aiwf history` which reads `git log` filtered by structured commit trailers.
---

# aiwf-history

The `aiwf history` verb answers "what happened to this entity?" by filtering `git log` for the entity's commit trailers. There is no separate event log; the git log is the time machine, made queryable by `aiwf-verb` / `aiwf-entity` / `aiwf-prior-entity` / `aiwf-to` / `aiwf-force` trailers.

## When to use

The user wants the lifecycle of one entity. Example phrasings: "when was M-007 created?", "show me what happened to E-19", "why is this gap closed?", "show the TDD cycle for M-007/AC-1".

## What to run

```bash
aiwf history <id>                    # one line per event
aiwf history <M-id>/AC-N             # composite id — just that AC's events
aiwf history <id> --format=json
```

The output is one event per line: `DATE  ACTOR  VERB  TO  DETAIL  COMMIT`. The TO column shows the target status/phase from the `aiwf-to:` trailer (`→ active`, `→ green`, etc.) or a dash for events with no target (add, rename, cancel — and pre-I2 promote commits whose schema didn't include `aiwf-to:`). Forced transitions are flagged with a `[forced: <reason>]` line beneath the main row.

## Composite ids and prefix matching

- `aiwf history M-007` shows the milestone's own events PLUS every AC's events (`M-007/AC-N`). The match is anchored on the literal `/` boundary so `M-007/` cannot prefix-match `M-070/`.
- `aiwf history M-007/AC-1` shows only that AC's events.

## What aiwf does

1. Runs `git log` with greps for `aiwf-entity: <id>` OR `aiwf-prior-entity: <id>` trailers (so reallocate events surface from both ids). For bare milestone ids, additionally greps for `<id>/AC-\d+` so the milestone view includes its ACs.
2. Parses each matching commit's subject, structured trailers (`aiwf-verb`, `aiwf-actor`, `aiwf-to`, `aiwf-force`), and author date.
3. Renders one event per line; forced events get an indented `[forced: <reason>]` line.

## Limitations

- `aiwf history` shows only verb-driven events. Hand-edits to the markdown file won't have trailers and will not appear. To see byte-level history of a file, use `git log -- <path>`.
- After a reallocate, query both ids if you want the full picture; the new id's history starts from the reallocate event.
- Pre-I2 promote commits don't carry `aiwf-to:`; the column renders as a dash. No retroactive fill.

## Don't

- Don't try to reconstruct history from filesystem timestamps — `git log` is authoritative.
- Don't expect prose-body changes to show up. Only frontmatter mutations through aiwf verbs are queryable here.
