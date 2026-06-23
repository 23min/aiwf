---
id: M-0172
title: area-unknown check finding for undeclared area values
status: draft
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: Declared area produces no finding
      status: open
      tdd_phase: red
    - id: AC-2
      title: Undeclared area fires area-unknown naming id, value, and set
      status: open
      tdd_phase: red
    - id: AC-3
      title: Absent, empty, or null area never fires
      status: open
      tdd_phase: red
    - id: AC-4
      title: Inert when no areas block is declared
      status: open
      tdd_phase: red
    - id: AC-5
      title: Archived entities never fire
      status: open
      tdd_phase: red
    - id: AC-6
      title: Finding code carries a hint and is discoverable
      status: open
      tdd_phase: red
---
## Goal

Add the `area-unknown` `aiwf check` finding: the present-⇒-declared chokepoint. When an entity's `area` is present and non-empty but not in the `aiwf.yaml: areas` member set, the check flags it (typo protection). Absence is never evaluated, and the rule is inert when no `areas` block exists.

## Context

M-0171 makes the `area` field and the `aiwf.yaml: areas` block exist and parse, but deliberately does not validate an entity's area against the declared set. This milestone adds that validation as a check rule — the authoritative surface (a creation-time flag alone can't catch a hand-edit or an `aiwf import` that introduces an undeclared area), mirroring the defense-in-depth pattern G-0268's `milestone-tdd-undeclared` follows.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac` against this milestone.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — An entity whose `area` is in the declared set produces no finding.
- **AC-2 candidate** — An entity whose `area` is present, non-empty, and *not* in the declared set produces an `area-unknown` finding naming the entity id, the offending value, and the declared set.
- **AC-3 candidate** — An entity with no `area`, empty `area`, or explicit null never produces the finding (absence is never evaluated).
- **AC-4 candidate** — With no `areas` block in `aiwf.yaml`, the rule is inert (no findings regardless of entity `area` values).
- **AC-5 candidate** — Archive-scoped per ADR-0004 §"`aiwf check` shape rules" (archived entities don't fire), consistent with the other shape-and-health rules.
- **AC-6 candidate** — The finding code is discoverable (aiwf-check skill row) and carries a hint; the three finding-code policies (tests, discoverability, hints) pass.

## Constraints

- **Single source of truth** for the declared set is `aiwf.yaml: areas` — the same accessor M-0171 introduces; no parallel reader.
- Default severity is a project decision to settle during implementation (lean: warning, with escalation under an existing or new strictness knob only if real friction shows — do not invent a knob speculatively).

## Out of scope

- The `aiwf add --area` write path (separate milestone).
- Read-surface filtering or grouping.
- Any auto-correction of an unknown area — the finding reports; the operator fixes.

## Dependencies

- M-0171 — the `area` field and `aiwf.yaml: areas` block + accessor.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)
- G-0268's `milestone-tdd-undeclared` — the archive-scoped check-finding pattern this rule follows.

### AC-1 — Declared area produces no finding

### AC-2 — Undeclared area fires area-unknown naming id, value, and set

### AC-3 — Absent, empty, or null area never fires

### AC-4 — Inert when no areas block is declared

### AC-5 — Archived entities never fire

### AC-6 — Finding code carries a hint and is discoverable

