---
id: M-0195
title: rename-area preserves comments and sibling keys in the areas block on rename
status: in_progress
parent: E-0044
tdd: required
acs:
    - id: AC-1
      title: Surgical member rename preserves all other areas-block bytes
      status: open
      tdd_phase: red
    - id: AC-2
      title: rename-area preserves areas.required and inner areas-block comments end-to-end
      status: open
      tdd_phase: red
    - id: AC-3
      title: rename-area preserves an unmodeled areas sub-key (forward-compat)
      status: open
      tdd_phase: red
---
## Goal

Make `aiwf rename-area` rewrite **only** the renamed member's name token in `aiwf.yaml`,
preserving every other byte of the `areas:` block — comments, sibling keys (`default`,
`required`, and any future key such as M-0185's `coverage_roots`), `paths`, member form, and
formatting. Today the verb regenerates the whole block from structured data via
`aiwfyaml.marshalAreasBlock`, silently dropping anything that data does not carry.

## Context

`aiwf rename-area` is the only verb that rewrites the `areas:` block. It routes through
`aiwfyaml.Doc.SetAreas` → `marshalAreasBlock`, which emits only `members:` and `default:` and
byte-splices the regenerated block over the original via `replaceAreas`. Everything else inside
`areas:` is lost.

This is already a **live regression**: a 1:1 monorepo with `areas.required: true` silently
reverts to non-strict on the next `rename-area` (confirmed by repro — the `required: true` line
vanishes). The same mechanism would drop M-0185's `coverage_roots`, and it destroys operator
comments that document why an area or glob is shaped the way it is.

The root cause is whole-block regeneration. The fix is to stop regenerating: a rename changes
exactly one scalar (a member's name), so splice only that token's bytes and leave the rest
untouched. This makes the no-drop guarantee **structural** — the writer never touches keys or
comments it is not renaming, so it cannot drop them — and forward-proofs every future `areas:`
key without per-field writer maintenance.

Discovered during M-0185 preflight (the `coverage_roots` knob would otherwise have shipped with
the same silent-drop hole). Sequenced before M-0185, which depends on this fix.

## Acceptance criteria

### AC-1 — Surgical member rename preserves all other areas-block bytes

A new `aiwfyaml` operation renames one member by replacing only its name scalar's source bytes
(located via the yaml.v3 node position plus the existing `lineToByteOffset` infra, branching on
quote style to find the token's end). Output is byte-identical to input except the renamed token,
which is re-emitted through `yamlScalar` so a name that newly needs quoting is quoted. Covered by
a unit matrix: legacy string-form member, object-form member, an already-quoted name, a name that
newly needs quoting, a member line carrying an inline comment, inner block comments, and a sibling
`required:` key.

### AC-2 — rename-area preserves areas.required and inner areas-block comments end-to-end

Driving `aiwf rename-area` through the verb/CLI seam against a fixture whose `areas:` block
carries `required: true` and a comment inside the block: after the rename, `required: true` is
still present, the comment is preserved verbatim, the member is renamed, and every entity tagged
with the old name is retagged — all in one commit. Pins the seam (the verb actually exercises the
surgical path), not just the writer layer.

### AC-3 — rename-area preserves an unmodeled areas sub-key (forward-compat)

A fixture whose `areas:` block carries a key the current config schema does not model (e.g. a
pre-landing `coverage_roots:`) plus a comment: after a rename, the unmodeled key and the comment
survive byte-for-byte. config-load silently ignores unknown `areas:` keys, so this is loadable
today; the AC proves M-0185's `coverage_roots` will round-trip with **zero** writer changes — the
structural form of the drop-proofing guarantee.

## Constraints

- The surgical splice touches only the member-name token; the entity-frontmatter retag path (the
  other half of `rename-area`) is unchanged.
- A name re-emitted after rename is quoted via the existing `yamlScalar` / `needsQuoting` helpers
  — no second quoting policy.
- `aiwfyaml` stays zero-dependency on `config` (the rename operation takes primitives, not a
  `config.Areas`).
- One atomic commit per rename (unchanged); a failure writes nothing.

## Out of scope

- Whole-block *re-serialization* / canonical reformatting of `areas:` — the opposite of this
  milestone.
- Comment preservation in **entity** frontmatter rewrites (a different writer; not regressed here).
- Adding the `coverage_roots` knob itself — that is M-0185.
- A speculative guard against a *future* verb reintroducing block regeneration — no current verb
  needs it (YAGNI); AC-3 pins `rename-area` specifically.

## Design notes

- The dead `marshalAreasBlock` / `Doc.SetAreas` whole-block path is removed once `rename-area` no
  longer calls it; its tests are replaced by the surgical operation's matrix.
- Locating the token: the name scalar's yaml.v3 node gives the 1-based `Line` / `Column` (Column
  points at the value start, including any opening quote, identically for string-form and
  object-form members). Token end is found by quote style — plain scalars cut exactly
  `len(oldName)`; quoted scalars scan to the matching close quote.
- Member names are validated unique, so the located node is unambiguous.

## Dependencies

- None upstream. **M-0185 depends on this** (its `coverage_roots` rides on the fixed writer).

## References

- `internal/aiwfyaml/aiwfyaml.go` — `SetAreas`, `marshalAreasBlock`, `replaceAreas`,
  `lineToByteOffset`, `yamlScalar` (the writer this milestone reworks).
- `internal/verb/renamearea.go` / `internal/cli/renamearea/renamearea.go` — the verb and CLI seam
  (`cliutil.ConfiguredAreaRequired` already exists).
- M-0177 — the `rename-area` verb this hardens.
- M-0178 — the `areas.required` knob whose silent drop this fixes.
- M-0185 — the dependent scoped-coverage milestone (`coverage_roots`).
- G-0287 — the member-level strict-key guard (the read-side analog of this no-silent-loss principle).
