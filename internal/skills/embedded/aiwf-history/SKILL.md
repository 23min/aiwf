---
name: aiwf-history
description: Use when the user asks "what happened to <entity>" or wants the timeline of an aiwf entity (or acceptance criterion). Runs `aiwf history` which reads `git log` filtered by structured commit trailers.
---

# aiwf-history

The `aiwf history` verb answers "what happened to this entity?" by filtering `git log` for the entity's commit trailers. There is no separate event log; the git log is the time machine, made queryable by `aiwf-verb` / `aiwf-entity` / `aiwf-prior-entity` / `aiwf-to` / `aiwf-force` / `aiwf-audit-only` and the I2.5 provenance trailers (`aiwf-principal`, `aiwf-on-behalf-of`, `aiwf-authorized-by`, `aiwf-scope`, `aiwf-scope-ends`, `aiwf-reason`).

## When to use

The user wants the lifecycle of one entity. Example phrasings: "when was M-007 created?", "show me what happened to E-19", "why is this gap closed?", "show the TDD cycle for M-007/AC-1", "who authorized this work?".

## What to run

```bash
aiwf history <id>                            # one line per event
aiwf history <M-id>/AC-N                     # composite id — just that AC's events
aiwf history <id> --show-authorization       # expand the auth-SHA column inline
aiwf history <id> --format=json              # full trailer set in JSON
```

The output is one event per line:

```
DATE  ACTOR  VERB  TO  DETAIL  COMMIT  [chips...]
```

- **ACTOR**: when a `principal` is present and differs from the actor (the agent-acts-for-human pattern), the column renders `principal via agent`. Direct human acts show the actor verbatim.
- **TO**: target status/phase from `aiwf-to:` (`→ active`, `→ green`); dash when absent.
- **chips**: compact lifecycle markers appended after the SHA — `[scope: opened]` on `aiwf authorize` rows, `[<scope-entity> <auth-short>]` on scope-authorized agent verbs, `[<scope-entity> ended]` per scope ended by a terminal-promote.
- Sub-lines (indented): `[forced: <reason>]`, `[audit-only: <reason>]`, `[reason: <text>]` for the corresponding trailers, then any commit body prose.

## Composite ids and prefix matching

- `aiwf history M-007` shows the milestone's own events PLUS every AC's events (`M-007/AC-N`). The match is anchored on the literal `/` boundary so `M-007/` cannot prefix-match `M-070/`.
- `aiwf history M-007/AC-1` shows only that AC's events.

## --show-authorization

By default scope chips abbreviate the auth-SHA to 7 characters: `[E-03 4b13a0f]`. With `--show-authorization`, the full SHA is inlined: `[E-03 4b13a0fdeadbeef...]`. Useful when copy-pasting into another `aiwf` invocation that needs the full SHA. JSON output always carries the full SHA in `authorized_by`.

## What aiwf does

1. Loads the entity tree once and resolves the queried id through `prior_ids` lineage to its current canonical entity. The chain is the queried id, plus every id in the canonical entity's `prior_ids`, plus the canonical entity's current id when distinct from all of those.
2. Runs `git log` with greps for `aiwf-entity: <id>` OR `aiwf-prior-entity: <id>` trailers for every id in the chain (so a single query weaves pre-rename, rename, and post-rename commits into one timeline). For bare milestone ids, additionally greps for `<id>/AC-\d+` so the milestone view includes its ACs.
3. Parses each matching commit's subject, structured trailers, and author date.
4. Builds a one-time `authSHA → scope-entity` map for chip rendering.
5. Renders one event per line; forced/audit-only/reason events get indented sub-lines.

After two reallocates (G-001 → G-002 → G-003), `aiwf history G-001`, `aiwf history G-002`, and `aiwf history G-003` all return the same chronological timeline.

## Limitations

- `aiwf history` shows only verb-driven events. Hand-edits to the markdown file won't have trailers and won't appear here. `aiwf check` flags such commits as `provenance-untrailered-entity-commit` (warning) at push time so the audit gap is visible; `aiwf <verb> --audit-only --reason "..."` is the repair path.
- Pre-I2 promote commits don't carry `aiwf-to:`; the column renders as a dash. No retroactive fill.
- Pre-G37 reallocates (before `prior_ids` shipped) don't carry the lineage in frontmatter. Querying the old id still surfaces the rename event via the existing `aiwf-prior-entity:` trailer, but post-rename history surfaces only when querying the new id. New reallocates fill `prior_ids` automatically; legacy chains can be backfilled manually if needed.

## Don't

- Don't try to reconstruct history from filesystem timestamps — `git log` is authoritative.
- Don't expect prose-body changes to show up. Only frontmatter mutations through aiwf verbs are queryable here.
- Don't ignore a `[forced: ...]` or `[audit-only: ...]` chip — they signal a sovereign override and rarely come without context worth surfacing to the user.
