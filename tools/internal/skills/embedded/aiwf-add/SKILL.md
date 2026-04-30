---
name: aiwf-add
description: Use when the user wants to add a new aiwf entity (epic, milestone, ADR, gap, decision, or contract). Runs `aiwf add` so the id allocation, frontmatter, and commit happen mechanically.
---

# aiwf-add

The `aiwf add` verb creates a new entity and produces exactly one git commit. Skills like this one are advisory; the binary is authoritative.

## When to use

The user wants to record a new piece of planning state — a new epic, a milestone under an existing epic, an ADR, a discovered gap, a decision, or a contract.

## What to run

```bash
aiwf add <kind> --title "<title>" [kind-specific flags]
```

The six kinds and their required flags:

| Kind | Required flags | Notes |
|---|---|---|
| epic | `--title` | Allocates `E-NN`. |
| milestone | `--title`, `--epic <E-id>` | Lives under the epic's directory. |
| adr | `--title` | Allocates `ADR-NNNN` under `docs/adr/`. |
| gap | `--title` | Optional `--discovered-in <id>`. |
| decision | `--title` | Optional `--relates-to <id,id,...>`. |
| contract | `--title` | Allocates `C-NNN` and creates `work/contracts/C-NNN-<slug>/contract.md`. Schema and validator binding land later via `aiwf contract bind`. |

## What aiwf does

1. Allocates the next free id by scanning the tree.
2. Writes the new entity file with proper frontmatter (`id`, `title`, `status` set to the kind's initial status).
3. Validates the projected tree before touching disk; if a finding would be introduced, aborts with no changes.
4. Creates one commit carrying `aiwf-verb: add`, `aiwf-entity: <id>`, `aiwf-actor: <actor>` trailers.

## Don't

- Don't hand-edit frontmatter to "skip allocation" — the id allocator + commit trailer chain is what makes history queryable.
- Don't pre-create the directory; `aiwf add` does that.
- Don't pass `--actor` unless the user asked for a specific actor; the default (derived from `aiwf.yaml` or git config) is correct.
