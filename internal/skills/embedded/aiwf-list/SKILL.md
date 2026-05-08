---
name: aiwf-list
description: Use to filter the planning tree by kind, status, parent, or archive flag — answers prompts like "list every milestone with status X", "find all entities matching Y", "show me proposed ADRs under E-13", "filter milestones by parent epic", "every gap that's still open", "every contract", "every AC that's met". Runs `aiwf list` and returns one row per matching entity (or a per-kind count summary when called with no flags). Read-only; no commit.
---

# aiwf-list

The hot-path read primitive over the planning tree. Reach for this whenever the user wants entities filtered by a structured axis — *every milestone with status X under epic Y*, *all proposed ADRs*, *every open gap*, *every contract*. `aiwf list` answers the query; `aiwf status` answers narrative-shaped state questions like *"what's next?"*. They split the read surface deliberately.

## What it does

`aiwf list` walks the planning tree and emits one summary row per entity matching the supplied filters. Default semantic: only **non-terminal-status entities** appear (closed/done/cancelled/addressed/wontfix/rejected/superseded/retired are hidden). Pass `--archived` to widen.

V1 filter axes:

| Flag | Filter |
|---|---|
| `--kind <K>` | one of `epic`, `milestone`, `adr`, `gap`, `decision`, `contract` |
| `--status <S>` | a status valid under `--kind` (or any kind's status set if `--kind` is omitted) |
| `--parent <id>` | entities whose `parent:` field is this id (e.g., milestones under an epic) |
| `--archived` | include terminal-status entities (the default hides them) |
| `--format=text\|json` | output shape; `--pretty` indents JSON |

Sort: id ascending, always.

## When to use

When the user reaches for *filter*, *list*, *find*, *show all*, or names a structured query shape:

| User says | Run |
|---|---|
| "list every milestone with status `done` under E-13" | `aiwf list --kind milestone --status done --parent E-13` |
| "find all proposed ADRs" | `aiwf list --kind adr --status proposed` |
| "every open gap" | `aiwf list --kind gap --status open` |
| "every contract" | `aiwf list --kind contract` |
| "what milestones are draft right now?" | `aiwf list --kind milestone --status draft` |
| "any cancelled work?" | `aiwf list --status cancelled --archived` |
| "give me the per-kind counts" | `aiwf list` (no flags) |

For machine consumption: append `--format=json [--pretty]`. The envelope's `result` is an array of summary objects with `{id, kind, status, title, parent, path}`.

## Recipes

- **Per-kind summary** — `aiwf list` (no args). Prints `5 epics · 47 milestones · 12 ADRs · 14 gaps · 3 decisions · 1 contract` style line. Excludes terminal-status entities; that's the active surface only.
- **All milestones under an epic** — `aiwf list --kind milestone --parent E-13`. Drops `--status` to see every status; add `--status in_progress` to narrow.
- **Every open gap** — `aiwf list --kind gap --status open`. Same data the *Open gaps* slice in `aiwf status` shows; both routes share one filter helper.
- **Every contract entity** — `aiwf list --kind contract`. Pair with `aiwf show <C-id>` for the full record.
- **All terminal-status entities** — `aiwf list --archived`. The `--archived` name is locked: ADR-0004 (proposed) names this verbatim and once that ADR ships, the same flag walks the archive directories without a list-side change.
- **Pipe to tooling** — `aiwf list --kind milestone --status done --format=json --pretty | jq '.result[].id'`.

## Output

Default text: one tab-aligned row per entity with header.

```
ID      STATUS       TITLE                              PARENT
M-001   draft        prep and schema                    E-01
M-002   in_progress  auth wiring                        E-01
```

Empty result is printed as nothing — no empty header. Grep- and pipe-friendly.

JSON envelope:

```json
{
  "tool": "aiwf",
  "version": "<semver>",
  "status": "ok",
  "findings": [],
  "result": [
    {"id": "M-001", "kind": "milestone", "status": "draft", "title": "prep and schema", "parent": "E-01", "path": "work/epics/E-01-.../M-001-....md"}
  ],
  "metadata": {"root": "<abs path>", "count": 1}
}
```

For the no-args invocation, `result` is an object `{ "epic": N, "milestone": N, ... }` rather than an array — the per-kind count payload.

## When to use list vs. status

Both verbs read the planning tree; their *job* differs:

- **Use `aiwf list`** for query-shaped prompts: anything where the user gives you a structured filter (*"every X with status Y"*, *"under parent Z"*, *"of kind K"*). The result is a flat row set sorted by id; you process it programmatically or surface it to the user.
- **Use `aiwf status`** for narrative-shaped prompts: *"what's next?"*, *"where are we?"*, *"what's in flight?"*. The result is a curated snapshot (in-flight epics + their milestones + open decisions + open gaps + recent activity + health) intended for the human reader, not for a downstream filter.

When in doubt: if the user named a kind or a status filter, reach for `aiwf list`. If they asked an open-ended state question, reach for `aiwf status`.
