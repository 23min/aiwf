---
id: M-0262
title: 'Add the priority write surface: set-priority verb and add --priority'
status: draft
parent: E-0066
depends_on:
    - M-0261
tdd: required
---

# M-0262 — Add the priority write surface: set-priority verb and add --priority

## Goal

Give operators two trailered ways to set a gap's or decision's `priority`: a dedicated `aiwf set-priority <id> <level>` verb for changing it later, and a `--priority` flag on `aiwf add` for setting it at creation.

## Context

The field and its validation land in the field milestone; this milestone makes it writable through verb routes so a value gets in without hand-editing frontmatter (which trips `provenance-untrailered-entity-commit`). `set-priority` is a deliberate second member of a `set-X` family alongside `set-area`, not a general-purpose edit verb — the codebase has no generic "edit a frontmatter field" verb and isn't gaining one here.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

## Constraints

- `set-priority` follows the two-file verb pattern (`internal/verb` + `internal/cli/…`), emits `aiwf-verb` / `aiwf-entity` / `aiwf-actor` trailers, and refuses a no-op set.
- `aiwf add --priority` is gated on kind exactly the way `--area` already is — legal on gap/decision, refused elsewhere.
- Both writers route validation through the field milestone's closed-set predicate; neither re-implements the value check.

## Design notes

- Verb wiring must satisfy the discoverability chokepoints: a new `aiwf-set-priority` skill (per `skill_coverage.go`) and completion wiring for the `<level>` arg (per the completion-drift test), mirroring `set-area`'s `CompleteAreaValueArg`.
- The `aiwf-add` skill gains a `--priority` line; the new `aiwf-set-priority` skill documents the verb.

## Surfaces touched

- `internal/verb/`, `internal/cli/` — the `set-priority` verb and its cobra wiring; the `--priority` flag on `aiwf add`.
- `internal/skills/embedded/` — the new `aiwf-set-priority` skill and the `aiwf-add` update.

## Out of scope

- Reading or filtering by priority (the read-surface milestone) and rendering it (the render milestone).
- A general-purpose `aiwf set <id> --field=value` verb.

## Dependencies

- M-0261 — the field, closed-set predicate, and validation must exist first.

## References

- G-0078 — the ratified design decisions (verb choice, creation-time flag).
- The `set-area` verb — `internal/cli/setarea/` — the pattern this verb copies.
