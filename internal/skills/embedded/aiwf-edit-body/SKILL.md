---
name: aiwf-edit-body
description: Use when the user wants to edit (rewrite or replace) the markdown body of an existing entity — goal/scope/context prose, AC body sections inside a milestone, ADR rationale, gap problem statement, etc. Runs `aiwf edit-body` so the change rides through a verb route with proper trailers, instead of a plain `git commit` that triggers a `provenance-untrailered-entity-commit` warning.
---

# aiwf-edit-body

The `aiwf edit-body` verb replaces the markdown body of an existing entity in a single trailered commit. Frontmatter is left untouched — that stays the domain of `aiwf promote`, `aiwf rename`, `aiwf cancel`, and `aiwf reallocate`.

## When to use

The user wants to update an entity's body prose: flesh out goal/scope, rewrite an ADR's rationale, add detail under AC body sections (by editing the parent milestone), update a gap's problem statement, etc. Anything below the YAML frontmatter is fair game.

## What to run

The verb has two modes. Bless mode is the default and matches the natural human workflow: edit the entity file in your editor, then commit through the verb route.

```bash
# Bless mode (default — no --body-file). The user has already edited
# the entity file in $EDITOR; the verb commits whatever changed:
aiwf edit-body <id>
aiwf edit-body <id> --reason "<why>"

# Explicit-content mode (M-058 — body comes from a file or stdin).
# Useful when an LLM session, script, or pipeline produces the
# body content and you want to apply it without an editor round-trip:
aiwf edit-body <id> --body-file <path>
aiwf edit-body <id> --body-file -                # read from stdin
aiwf edit-body <id> --body-file <path> --reason "<why>"
```

Both modes refuse leading `---` (frontmatter delimiter) in body content — the verb is body-only, so a body file containing its own frontmatter would produce a malformed double-block file.

### Bless mode rules

- **No diff**: refuses with "no changes to commit" rather than producing an empty commit.
- **Frontmatter changed**: refuses and points at `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate`. Bless mode is body-only by design; structured-state edits go through their own verbs.
- **New entity (no HEAD version)**: refuses with a pointer to `aiwf add --body-file` for create-time body content.
- **YAML formatting preserved**: bless mode commits the working-copy bytes verbatim — key order, comments, and whitespace from the user's edit are not re-canonicalized through the loader. (Explicit mode does re-serialize through `entity.Serialize`, which canonicalizes.)

### AC body sub-sections

Editing the prose under a single `### AC-N — title` heading inside a milestone body works through bless mode on the parent milestone — edit the section in $EDITOR, run `aiwf edit-body M-NNN`. The verb commits whatever changed; no composite-id resolver needed. (Composite ids `M-NNN/AC-N` are still refused to keep the verb's seam simple.)

## What aiwf does

**Bless mode** (no `--body-file`):

1. Loads the entity by id, reads the working-copy bytes and the HEAD version of the file.
2. Refuses if there is no diff, the file has no HEAD version (new entity — use `aiwf add` instead), or the diff includes frontmatter changes.
3. Validates the working-copy body content (refuses leading `---`).
4. Writes one OpWrite of the working-copy bytes verbatim and creates one commit with `aiwf-verb: edit-body`, `aiwf-entity: <id>`, `aiwf-actor: <actor>`. `--reason "..."` lands in the commit body.

**Explicit-content mode** (`--body-file <path>` or stdin):

1. Loads the entity by id, validates the supplied body content (refuses leading `---`).
2. Re-serializes the entity with its existing frontmatter unchanged and the new body in place.
3. Writes one OpWrite to the entity file and creates one commit with the same trailer set as bless mode — `aiwf history` cannot tell them apart, which is the right outcome.

The frontmatter `id`, `title`, `status`, references, and acs[] are all preserved verbatim in both modes. Structured-state edits go through `aiwf promote` / `aiwf rename` / `aiwf cancel` / `aiwf reallocate`.

## Composite ids (M-NNN/AC-N)

Not directly supported. To update an AC body section, edit the prose under the `### AC-N — title` heading in the parent milestone file in your editor, then run `aiwf edit-body M-NNN` (bless mode commits whatever changed). Composite-id support is deliberately deferred — bless mode covers the AC sub-section workflow without needing a sub-section resolver.

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
