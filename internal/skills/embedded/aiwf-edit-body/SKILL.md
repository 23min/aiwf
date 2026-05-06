---
name: aiwf-edit-body
description: Use when the user wants to edit (rewrite or replace) the markdown body of an existing entity — goal/scope/context prose, AC body sections inside a milestone, ADR rationale, gap problem statement, etc. Runs `aiwf edit-body` so the change rides through a verb route with proper trailers, instead of a plain `git commit` that triggers a `provenance-untrailered-entity-commit` warning.
---

# aiwf-edit-body

The `aiwf edit-body` verb replaces the markdown body of an existing entity in a single trailered commit. Frontmatter is left untouched — that stays the domain of `aiwf promote`, `aiwf rename`, `aiwf cancel`, and `aiwf reallocate`.

## When to use

The user wants to update an entity's body prose: flesh out goal/scope, rewrite an ADR's rationale, add detail under AC body sections (by editing the parent milestone), update a gap's problem statement, etc. Anything below the YAML frontmatter is fair game.

## What to run

```bash
aiwf edit-body <id> --body-file <path>           # read body from a file
aiwf edit-body <id> --body-file -                # read body from stdin (pipe)
aiwf edit-body <id> --body-file <path> --reason "<why>"  # optional commit-body rationale
```

The body file contains markdown only — no YAML frontmatter. A leading `---` (frontmatter delimiter) is refused so the rewrite can't accidentally produce a double-frontmatter file.

## What aiwf does

1. Loads the entity by id, validates the supplied body content (refuses leading `---`).
2. Re-serializes the entity with its existing frontmatter unchanged and the new body in place.
3. Validates the projected tree before touching disk.
4. Writes one OpWrite to the entity file and creates one commit carrying `aiwf-verb: edit-body`, `aiwf-entity: <id>`, `aiwf-actor: <actor>`. `--reason "..."` lands in the commit body so `aiwf history` surfaces the rationale.

The frontmatter `id`, `title`, `status`, references, and acs[] are all preserved verbatim. If the user wants to update structured state, that's a different verb (promote, rename, cancel).

## Composite ids (M-NNN/AC-N)

Not yet supported. To update an AC body section, edit the parent milestone's body — the AC heading (`### AC-N — title`) and its prose live there. A future verb may target sub-sections directly; for now, scope edits to the whole milestone body when an AC's prose needs work.

## Provenance flags

| Flag | When |
|---|---|
| `--actor <role>/<id>` | Override the runtime-derived identity (default: `human/<localpart-of-git-config-user.email>`). |
| `--principal human/<id>` | **Required** when `--actor` is non-human (`ai/...`, `bot/...`); **forbidden** when `--actor` is `human/...`. |

Agents acting under an active authorization scope get scope trailers stamped automatically; without an active scope, agent invocations refuse with `provenance-no-active-scope`.

## Don't

- Don't hand-edit frontmatter through this verb's body file — frontmatter is structurally separate; use `aiwf promote` / `aiwf rename` / `aiwf cancel` instead.
- Don't include a frontmatter block in the body file — the verb refuses to prevent malformed output.
- Don't use `aiwf edit-body` for status transitions or renames; those have their own verbs and proper FSM/atomicity rules.
