---
id: M-0181
title: Mistag detection via aiwf-entity trailer with acknowledge path
status: in_progress
parent: E-0044
depends_on:
    - M-0179
tdd: required
acs:
    - id: AC-1
      title: Gather an entity's commits and touched paths via the aiwf-entity trailer
      status: met
      tdd_phase: done
    - id: AC-2
      title: area-mistag fires when all area-claimed work lands in foreign areas
      status: met
      tdd_phase: done
    - id: AC-3
      title: No finding when some touched paths land in the entity's own area
      status: open
      tdd_phase: red
    - id: AC-4
      title: Inert with no paths, no linked commits, global area, or archived entity
      status: open
      tdd_phase: red
    - id: AC-5
      title: Regroup acknowledge-illegal into the aiwf acknowledge illegal subverb
      status: open
      tdd_phase: red
    - id: AC-6
      title: aiwf acknowledge mistag records a sovereign ack the check suppresses
      status: open
      tdd_phase: red
    - id: AC-7
      title: area-mistag and the acknowledge surface are discoverable and pinned
      status: open
      tdd_phase: red
---
## Goal

Flag a landed entity whose commits touch only another area's paths: gather the entity's commits via the `aiwf-entity:` trailer, intersect the touched files with the entity's area glob, and warn when the diff falls entirely outside it — with a sovereign-traced acknowledge path for legitimate cross-cutting work.

## Context

This is the check that actually catches "filed against the wrong area, flew under the radar" — the failure label-only areas are blind to. With `paths:` (M-0179) and the entity ↔ commit linkage aiwf already records via trailers, the touched-files-vs-glob comparison becomes buildable.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- For a landed entity, the check gathers its commits via the `aiwf-entity:` trailer and computes the set of touched paths.
- A warning fires when every touched path falls outside the entity's area glob.
- No finding when at least some touched paths fall inside the area glob — cross-cutting is tolerated, not policed.
- An operator can acknowledge a flagged entity via a named, reasoned act (human actor + `--reason`), mirroring `aiwf acknowledge-illegal`; the acknowledgement suppresses the finding for that entity.
- Inert when no `paths:` are declared, and for entities with no linked commits (planned work with no diff).

## Constraints

- Warning severity, never gating; legitimate cross-cutting exists.
- Acknowledge is sovereign-traced (human actor, written reason), per the provenance model.

## Out of scope

- Auto-correcting the tag — suggestion / derivation is the auto-derive milestone.

## Dependencies

- M-0179 (`paths:` per area) — the oracle the diff is checked against.

## References

- `aiwf acknowledge-illegal` skill — the acknowledge-with-reason precedent.
- The `aiwf-entity:` commit trailer — the entity ↔ commit linkage this reads.

### AC-1 — Gather an entity's commits and touched paths via the aiwf-entity trailer

### AC-2 — area-mistag fires when all area-claimed work lands in foreign areas

### AC-3 — No finding when some touched paths land in the entity's own area

### AC-4 — Inert with no paths, no linked commits, global area, or archived entity

### AC-5 — Regroup acknowledge-illegal into the aiwf acknowledge illegal subverb

### AC-6 — aiwf acknowledge mistag records a sovereign ack the check suppresses

### AC-7 — area-mistag and the acknowledge surface are discoverable and pinned

