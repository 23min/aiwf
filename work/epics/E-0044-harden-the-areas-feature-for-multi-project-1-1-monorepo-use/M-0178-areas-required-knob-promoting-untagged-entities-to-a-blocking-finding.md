---
id: M-0178
title: areas.required knob promoting untagged entities to a blocking finding
status: draft
parent: E-0044
depends_on:
    - M-0183
tdd: required
acs:
    - id: AC-1
      title: areas.required parses as a bool; required with zero members is rejected
      status: open
      tdd_phase: red
    - id: AC-2
      title: area-required errors on every untagged root entity across all five kinds
      status: open
      tdd_phase: red
    - id: AC-3
      title: required off or absent leaves area-required inert (pre-knob parity)
      status: open
      tdd_phase: red
    - id: AC-4
      title: a milestone never fires area-required; an untagged epic reports once
      status: open
      tdd_phase: red
    - id: AC-5
      title: aiwf add refuses an untagged create when areas.required is true
      status: open
      tdd_phase: red
    - id: AC-6
      title: area-required ships SKILL.md discoverability and a set-area hint
      status: open
      tdd_phase: red
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

### AC-1 — areas.required parses as a bool; required with zero members is rejected

### AC-2 — area-required errors on every untagged root entity across all five kinds

### AC-3 — required off or absent leaves area-required inert (pre-knob parity)

### AC-4 — a milestone never fires area-required; an untagged epic reports once

### AC-5 — aiwf add refuses an untagged create when areas.required is true

### AC-6 — area-required ships SKILL.md discoverability and a set-area hint

