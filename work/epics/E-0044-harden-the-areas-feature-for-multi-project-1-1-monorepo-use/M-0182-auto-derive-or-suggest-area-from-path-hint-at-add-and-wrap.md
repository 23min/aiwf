---
id: M-0182
title: Auto-derive or suggest area from path hint at add and wrap
status: draft
parent: E-0044
depends_on:
    - M-0179
tdd: required
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
