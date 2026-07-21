# Import manifest format

`aiwf import <manifest>` is the bulk entity-creation verb. It reads a declarative list of entities and writes them to the consumer tree in a single atomic commit. It is the public contract by which any external tool — a custom script, an export from another planning system, a generator — hands work to aiwf without aiwf needing to know anything about the source.

This document defines the format and the semantics. It does not define how a manifest is produced; that is the producer's concern.

For the engineering rationale that motivates this shape (declarative-only, no transformations, single atomic write), see [`design-decisions.md`](../design/design-decisions.md).

---

## What the manifest *is*

A list of entities to materialize. Each entry carries a kind, an id (or a request to allocate one), a frontmatter block, and a body. After import, those entities exist in canonical aiwf v3 form — same shape as if they had been authored by hand.

## What the manifest is *not*

- **Not a script.** No steps, no operations, no transformations. Each entry is a fully resolved declaration of a target entity.
- **Not a transformation language.** No regex, no templates, no expressions, no callbacks. Producers do whatever transformation they need *before* writing the manifest. The manifest carries data only.
- **Not a filter.** No "ignore" or "skip" entries. If a producer cannot or should not include an item, it omits it. Filtering at consumption time is a CLI concern (`--kind`, `--ids`), not a manifest field.

These exclusions are load-bearing. They keep the import surface small and keep aiwf unaware of the producer's internals.

---

## Format

YAML is the primary representation. JSON with the same field names is also accepted. The top-level shape:

```yaml
version: 1
actor: human/peter           # optional; defaults to git config user.email mapping
commit:
  mode: single               # single | per-entity   (default: single)
  message: "import: bulk entity creation"   # optional; default supplied
entities:
  - kind: epic
    id: E-11
    frontmatter:
      title: "Svelte UI"
      status: active
    body: |
      ## Goal

      Build a SvelteKit app in parallel with the Blazor frontend.

      ## Scope

      ...

  - kind: milestone
    id: M-001
    frontmatter:
      title: "Project scaffold"
      status: done
      parent: E-11
    body: |
      ## Goal

      Standing SvelteKit app with sidebar layout.

      ## Acceptance criteria

      - [x] `pnpm dev` serves the app
      - [x] Theme toggle persists

  - kind: decision
    id: auto                  # allocate next available D-NNN
    frontmatter:
      title: "Use pnpm for the UI workspace"
      status: accepted
    body: |
      ## Question
      ...
```

Five top-level concepts: `version`, `actor`, `commit`, `entities`, and the per-entity fields. Adding anything beyond this is a kernel decision.

---

## Top-level fields

| Field | Required | Type | Notes |
|---|---|---|---|
| `version` | yes | int | Manifest schema version. Currently `1`. |
| `actor` | no | string | The actor recorded in commit trailers. Defaults to the value `aiwf init` would derive from `git config user.email`. |
| `commit.mode` | no | enum | `single` (default) — one commit for the whole batch. `per-entity` — one commit per entity, in entity order. |
| `commit.message` | no | string | Override the default commit message (single-mode) or per-entity template (per-entity mode). |
| `entities` | yes | list | The entities to create. May be empty (no-op import). |

---

## Per-entity fields

| Field | Required | Type | Notes |
|---|---|---|---|
| `kind` | yes | enum | One of `epic`, `milestone`, `adr`, `gap`, `decision`, `contract`. |
| `id` | yes | string or `auto` | Explicit id matching the kind's regex (e.g. `E-11`, `M-001`), or the literal string `auto` to allocate the next available id of that kind. |
| `frontmatter` | yes | map | YAML frontmatter for the entity. Must satisfy the kind's required fields and pass `frontmatter-shape` and `status-valid` checks. |
| `body` | no | string | Markdown body. May be empty. Content is opaque to aiwf — written as-is. |

The `frontmatter` map is verbatim what the entity's YAML frontmatter will contain. Required and optional fields per kind are defined in the standard entity schemas; the import does not extend or relax them.

---

## Semantics

### ID allocation

- Entries with explicit ids are processed first, in manifest order. Their ids are reserved against the projected tree.
- Entries with `id: auto` are then allocated `max(reserved ∪ existing) + 1` per kind, in manifest order.
- This makes allocation deterministic given a fixed manifest and tree state.

### Reference resolution

- Reference fields (e.g. milestone `parent`, ADR `supersedes`) resolve against the union of (a) ids already in the consumer tree and (b) ids declared in the manifest, including ones being allocated as `auto`.
- `auto` ids are resolved in two passes: first an allocation pass (which assigns concrete ids to every `auto` entry), then a reference-resolution pass against the fully-allocated set.
- Forward and backward references within the manifest are both legal.

### Atomicity

- The import builds a projected tree (current state plus manifest entries) entirely in memory.
- `aiwf check` runs against the projection. If it produces any error-severity findings, the import aborts. No files are written.
- If the projection is clean, files are written and one commit (or one per entity, in `per-entity` mode) is produced.
- A failed import leaves the working tree and index untouched.

### Collisions

- An explicit id that already exists in the tree is, by default, an error. The import aborts.
- `--on-collision=skip` — skip manifest entries whose ids already exist; import the rest.
- `--on-collision=update` — overwrite frontmatter and body of existing entities. Body is replaced wholesale; no merging.
- `--on-collision=fail` — default. Abort the entire import.

`auto` allocation never collides with existing ids by construction; it allocates above the current maximum.

### Dry-run

- `aiwf import --dry-run <manifest>` runs the full validation pipeline, prints the diff that would be written and any findings, then exits without writing or committing.
- This is the primary mechanism for iterating on a manifest. Producers should expect to dry-run repeatedly and adjust before a real import.

### Commit trailers

- Single-commit mode: one commit with `aiwf-verb: import`, `aiwf-actor: <actor>`. No `aiwf-entity:` trailer (the commit creates many entities; no single id).
- Per-entity mode: one commit per entity with `aiwf-verb: add`, `aiwf-entity: <id>`, `aiwf-actor: <actor>`. Identical to invoking `aiwf add` per entity, but in one batch.

`aiwf history <id>` finds per-entity-mode commits via `aiwf-entity:`. Single-commit-mode imports are not addressable via `aiwf history`; use `git log --grep='aiwf-verb: import'` to find the import commit, then read the diff.

---

## Exit codes

Same as other aiwf verbs:

- `0` — import succeeded (or dry-run was clean).
- `1` — import aborted due to validation findings.
- `2` — usage error (missing manifest, malformed YAML, unknown field).
- `3` — internal error.

JSON output (`--format=json`) is structurally identical to `aiwf check` output: an envelope with findings keyed by code and severity, plus an `import` summary section listing the entries that would be created.

---

## Design notes

### Why declarative-only

A manifest that carried imperative steps would require aiwf to *interpret* the producer's intent. Once aiwf understands the producer's operations, the producer's conventions leak into aiwf. Keeping the manifest declarative makes aiwf a pure consumer of canonical data; producers do all transformation upstream.

### Why one atomic commit by default

Bulk creation under per-entity commits produces N commits in a single push, each individually valid but cluttering history. Single-commit mode is the right default for genuine batch operations (initial population of a tree, bulk import from another planning system). Per-entity mode is available when downstream tooling expects per-entity commit granularity.

### Why no "skip" or "ignore" in the manifest

A skipped item is just an item the producer chose not to emit. The producer's logs are the right place to record skip reasons; the manifest is the right place to record what *should* exist. Mixing the two would force aiwf to model "things that aren't entities," which has no other purpose in the framework.

### Why no transformations

The manifest is read once, validated, and written. If aiwf supported in-flight transformations (templates, expressions, regex), it would need to model an evaluation environment, error handling for transformation failures, and security boundaries on what transformations can do. None of that is in scope. Producers transform; aiwf writes.

---

## Producer responsibilities

A tool that produces an import manifest is responsible for:

- Resolving every value in the entities to its final form. No placeholders, no partial fields, no references to source-system identifiers.
- Choosing explicit ids vs. `auto` per entity. Explicit ids are appropriate when the producer wants stable referencing; `auto` is appropriate for greenfield creation where allocation is left to aiwf.
- Producing a separate human-readable report of what was *not* included and why. The manifest itself does not carry skip metadata.
- Iterating against `aiwf import --dry-run` until findings are clean. The producer-side and aiwf-side validation are two pressure points; a clean dry-run is the gate for a real import.
