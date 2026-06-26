---
id: M-0178
title: areas.required knob promoting untagged entities to a blocking finding
status: draft
parent: E-0044
depends_on:
    - M-0183
tdd: required
---
## Goal

Add an `areas.required: true` knob that makes an untagged entity illegal — a blocking `aiwf check` finding — for the 1:1 monorepo where every entity belongs to exactly one project. Inert (exactly E-0043's behavior) when absent or false.

## Context

E-0043 deliberately never flags an absent `area` ("absence is its own partition"). That is right for the carved-into-sections case but wrong for the 1:1 monorepo, where untagged is genuinely unassigned. This milestone adds the opt-in strictness without disturbing the default. It is orthogonal to `area-unknown` (which polices present ⇒ declared); this polices present-at-all.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- `aiwf.yaml: areas.required` (bool, default false) parses and validates.
- With `required: true` and a declared `areas` block, a non-archived entity with an empty `area` raises a blocking finding naming the entity.
- With `required` absent or false, behavior is byte-for-byte E-0043 (no finding for an absent area).
- The finding is a distinct code from `area-unknown`; milestones/ACs derive their area from the parent epic, so no double-report under an untagged epic.

## Constraints

- Default-off, zero migration: an existing tree with no `required` key validates and renders exactly as today.
- Does not gate the default views — `required` makes untagged a check finding, it does not make grouping hide anything.

## Out of scope

- Path verification (Tier 1) — `required` only asserts presence, not correctness against paths.
- Reusing or escalating `area-unknown` — that finding stays present ⇒ declared; this is a separate rule.

## Dependencies

- None. Independent Tier-0; parallel with the other Tier-0 milestones.

## References

- `internal/check/area_unknown.go` — the sibling present ⇒ declared finding this sits beside (not modified).
- `internal/config/config.go` — the `Areas` schema the knob extends.
