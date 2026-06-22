---
id: M-0171
title: Area field on root kinds and aiwf.yaml areas block with validation
status: draft
parent: E-0043
tdd: required
acs:
    - id: AC-1
      title: Five root kinds accept optional area frontmatter field; absent parses clean
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf.yaml areas block declares member set + optional default label, validated
      status: open
      tdd_phase: red
    - id: AC-3
      title: Milestone and AC derive area from parent epic at load, exposed in model
      status: open
      tdd_phase: red
    - id: AC-4
      title: 'With no areas block the area field is inert: parses but nothing validates'
      status: open
      tdd_phase: red
---
## Goal

Add the optional `area` frontmatter field to the five root entity kinds (epic, ADR, gap, decision, contract) and the `aiwf.yaml: areas` block that declares the closed member set. This is the data + config foundation the rest of E-0043 builds on; the flat, globally-unique id space is untouched.

## Context

Per E-0043's converged design, `area` is a validated grouping tag, not a directory axis or an id-space change. This milestone makes the field *exist and parse* and the config block *exist and validate* — it does not yet add the `area-unknown` check finding (next milestone), the write path, or any read surface. Until the `areas` block is present the field is inert.

Milestones and ACs do **not** store `area`; they derive it from their parent epic, so "milestone disagrees with its epic" is unrepresentable rather than policed.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac` against this milestone.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — The five root kinds (epic, ADR, gap, decision, contract) accept an optional `area:` string field in frontmatter; absent/empty parses cleanly (no error, no default written).
- **AC-2 candidate** — `aiwf.yaml` accepts an `areas` block: a closed member set plus an optional `default:` key that is a display label only (never a member, never written to an entity). Schema validation rejects a malformed block (non-string members, etc.) at config-load time.
- **AC-3 candidate** — A milestone (and an AC) resolves its `area` by deriving from its parent epic at load time — not stored on the milestone — exposed through the loaded model so downstream read surfaces can group without re-deriving.
- **AC-4 candidate** — With no `areas` block in `aiwf.yaml`, the `area` field is inert: present values parse but nothing validates or groups (validation lands as the `area-unknown` finding in the next milestone).
- **AC-5 candidate** — Strict-decoder forward-compat is documented at the field site (a pre-`area` binary rejects a file using it — the generic `KnownFields(true)` window, not special to `area`).

## Constraints

- **Commitment #2 (stable flat ids) untouched.** No change to the allocator, references, trailers, `aiwf history`, or `reallocate`. `area` never reshapes the on-disk tree, so the loader and the ADR-0004 archive convention are untouched.
- **Single source of truth** for the member set is `aiwf.yaml: areas`; no parallel registry.
- **Zero migration.** Every existing entity (no `area`) keeps parsing and rendering exactly as today.

## Out of scope

- The `area-unknown` check finding (next milestone).
- The `aiwf add --area` write path and completion (later milestone).
- Any read-surface filter or grouping (later milestones).

## Dependencies

- E-0043 epic spec (committed). No prior milestones — this is the foundation.

## References

- [E-0043 epic](epic.md) — converged design and scope.
- [G-0266](../../gaps/G-0266-optional-area-tag-for-grouping-entities-by-workstream.md) — the gap this epic implements.

### AC-1 — Five root kinds accept optional area frontmatter field; absent parses clean

### AC-2 — aiwf.yaml areas block declares member set + optional default label, validated

### AC-3 — Milestone and AC derive area from parent epic at load, exposed in model

### AC-4 — With no areas block the area field is inert: parses but nothing validates

