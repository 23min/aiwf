---
name: aiwf-promote
description: Use when the user wants to advance an entity to a new status. Runs `aiwf promote` so the transition is checked against the kind's legal moves and recorded as a single commit.
---

# aiwf-promote

The `aiwf promote` verb edits an entity's `status` field. Allowed transitions are hardcoded per kind; illegal moves are refused before any disk change.

## When to use

The user says something is "ready", "done", "in progress", "accepted", "deprecated", etc. — i.e. wants to move an entity from one status to another.

## What to run

```bash
aiwf promote <id> <new-status>
```

## Allowed status sets

| Kind | Statuses |
|---|---|
| epic | `proposed`, `active`, `done`, `cancelled` |
| milestone | `draft`, `in_progress`, `done`, `cancelled` |
| adr | `proposed`, `accepted`, `superseded`, `rejected` |
| gap | `open`, `addressed`, `wontfix` |
| decision | `proposed`, `accepted`, `superseded`, `rejected` |
| contract | `draft`, `published`, `deprecated`, `retired` |

`aiwf promote` enforces the per-kind legal-transition function. If the move is illegal it reports a finding and exits without writing. To reach a terminal-cancel status use `aiwf cancel <id>` instead — same end state, clearer intent in the log.

## What aiwf does

1. Loads the entity by id, validates the transition.
2. Rewrites only the `status:` line in frontmatter (everything else preserved).
3. Commits with trailers `aiwf-verb: promote`, `aiwf-entity: <id>`, `aiwf-actor: <actor>`.

## Don't

- Don't hand-edit `status:` in markdown — the trailer chain disappears and `aiwf history` won't surface the move.
- Don't try to skip statuses (e.g., `proposed` → `done` for an epic). The legal-transition function won't allow it; that's intentional.
