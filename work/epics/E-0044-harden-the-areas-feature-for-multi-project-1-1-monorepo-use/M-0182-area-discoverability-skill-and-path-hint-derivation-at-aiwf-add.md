---
id: M-0182
title: Area discoverability skill and path-hint derivation at aiwf add
status: draft
parent: E-0044
depends_on:
    - M-0179
tdd: required
acs:
    - id: AC-1
      title: Topical aiwf-area skill exists with valid frontmatter and is discoverable
      status: open
      tdd_phase: red
    - id: AC-2
      title: 'Skill teaches the area mental model: operate-everywhere vs aiwf constraints'
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'Skill teaches the area lifecycle: add, set-area, mistag, acknowledge'
      status: open
      tdd_phase: red
    - id: AC-4
      title: Single unambiguous --path-hint derives area when --area is omitted
      status: open
      tdd_phase: red
    - id: AC-5
      title: Explicit --area always wins over a conflicting --path-hint
      status: open
      tdd_phase: red
    - id: AC-6
      title: Ambiguous --path-hint sets no area, prints a suggestion, proceeds untagged
      status: open
      tdd_phase: red
    - id: AC-7
      title: Inert with no declared paths; areamatch.Derive is the SSOT primitive
      status: open
      tdd_phase: red
---
## Goal

Once areas know their paths, let `aiwf add` (and wrap) derive or suggest an entity's `area` from a path hint, so an operator tags correctly without typing the area name — driving manual tags and mistags toward zero.

## Context

Manual tagging is the source of mistags. With `paths:` (M-0179) a single unambiguous path hint maps to exactly one area, so the kernel can fill the tag. Planned work has no diff yet, so derivation lands from an explicit target-path hint at add time, or at implementation / wrap once a diff exists.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- Given a single path hint that falls under exactly one area's glob, `aiwf add` derives `area` when `--area` is omitted.
- An explicit `--area` always wins over derivation.
- An ambiguous hint (matching zero or multiple areas) does not silently set `area` — it suggests or leaves untagged. The default-on-vs-suggest open question is resolved here.
- Inert when no `paths:` are declared.

## Constraints

- Never silently overwrite an explicit `--area`.
- Keep the human in the loop for ambiguous diffs — suggest, don't guess.

## Out of scope

- Retroactively re-tagging existing entities in bulk.

## Dependencies

- M-0179 (`paths:` per area) — the oracle derivation reads.

## References

- The `aiwf add --area` write path (E-0043 / M-0173) — extended here with derivation.

### AC-1 — Topical aiwf-area skill exists with valid frontmatter and is discoverable

### AC-2 — Skill teaches the area mental model: operate-everywhere vs aiwf constraints

### AC-3 — Skill teaches the area lifecycle: add, set-area, mistag, acknowledge

### AC-4 — Single unambiguous --path-hint derives area when --area is omitted

### AC-5 — Explicit --area always wins over a conflicting --path-hint

### AC-6 — Ambiguous --path-hint sets no area, prints a suggestion, proceeds untagged

### AC-7 — Inert with no declared paths; areamatch.Derive is the SSOT primitive

