---
id: M-0174
title: --area filter on list, show, and status
status: in_progress
parent: E-0043
depends_on:
    - M-0171
tdd: required
acs:
    - id: AC-1
      title: list --area returns only entities whose effective area matches
      status: open
      tdd_phase: red
    - id: AC-2
      title: status --area scopes epics, decisions, and gaps to one area
      status: open
      tdd_phase: red
    - id: AC-3
      title: show --area shows the entity only when its effective area matches
      status: open
      tdd_phase: red
    - id: AC-4
      title: --area tab-completes the declared areas.members on list/show/status
      status: open
      tdd_phase: red
    - id: AC-5
      title: an undeclared --area value prints a note and yields an empty result
      status: open
      tdd_phase: red
    - id: AC-6
      title: untagged entities are excluded from a specific --area filter
      status: open
      tdd_phase: red
---
## Goal

Add an `--area <name>` filter to the read verbs `list`, `show`, and `status`, so an operator can scope each to a single workstream. Entities whose effective area (explicit for root kinds, parent-derived for milestones/ACs) matches the flag are shown; others are hidden.

## Context

M-0171 exposes each entity's effective `area` through the loaded model. This milestone consumes that for read-time scoping — the first half of "roadmaps/status/checks become scopeable per workstream." It is independent of the write-path milestone (it filters whatever is already tagged) and of the grouping milestone (filter narrows; grouping partitions).

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac` against this milestone.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — `aiwf list --area <name>` returns only entities whose effective area equals `<name>` (root kinds by explicit field; milestones/ACs by parent-derived area).
- **AC-2 candidate** — `aiwf status --area <name>` scopes the snapshot (in-flight epics, milestones, open items) to that workstream.
- **AC-3 candidate** — `aiwf show` honors `--area` where it lists multiple entities (or documents that show is single-entity and the flag is a no-op / rejected there — decided at implementation).
- **AC-4 candidate** — `--area` tab-completes the declared `areas` members (same wiring as the write-path flag); the completion-drift policy passes.
- **AC-5 candidate** — Filtering by an undeclared `--area` value behaves predictably (lean: empty result + a one-line note, not a silent empty), decided at implementation.
- **AC-6 candidate** — Untagged entities are excluded from a specific `--area` filter and surface only under the default complement (a representation question shared with the grouping milestone).

## Constraints

- **Read-only.** No mutation, no commit. `--area` is a view filter.
- **Effective-area is computed once** in the loaded model (M-0171), not re-derived per verb — single source of truth.

## Out of scope

- Area *grouping* (sectioned output) — that's the grouping milestone; this milestone only *filters*.
- The write path and the check finding.

## Dependencies

- M-0171 — effective-area exposure on the loaded model.

## References

- [E-0043 epic](epic.md) · [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md)

### AC-1 — list --area returns only entities whose effective area matches

### AC-2 — status --area scopes epics, decisions, and gaps to one area

### AC-3 — show --area shows the entity only when its effective area matches

### AC-4 — --area tab-completes the declared areas.members on list/show/status

### AC-5 — an undeclared --area value prints a note and yields an empty result

### AC-6 — untagged entities are excluded from a specific --area filter

