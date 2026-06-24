---
id: M-0180
title: Area path bijection and coverage check
status: draft
parent: E-0044
depends_on:
    - M-0179
tdd: required
---
## Goal

Add an `aiwf check` finding that verifies the area ↔ project-directory bijection: every declared area's glob matches a real directory (no dead config), and every project directory maps to exactly one area (no unslotted project, no overlap).

## Context

Once `paths:` exists (M-0179), the kernel can check the config against the filesystem. This catches two monorepo failure modes label-only areas can't see: a renamed or missing project directory leaving a dead glob, and a newly-added project nobody slotted into an area.

## Acceptance criteria

<!-- Candidate ACs, formalized via `aiwf add ac <id> --title "..."` at start-milestone. -->

Candidate behaviors to formalize at start-milestone:

- A declared area whose glob matches no directory raises a finding (dead config).
- A project directory matching no declared area's glob raises a finding (unslotted project) — the monorepo-specific reverse check.
- A directory matching more than one area's glob raises a finding (overlap).
- Severity per the deferred open question (lean: warning by default, blocking under `areas.required`).
- Inert when no `paths:` are declared — a string-form / label-only config never fires this.

## Constraints

- Reads the filesystem read-only; never writes. Composed at the CLI layer with the declared set sourced from config, like `area-unknown`.
- Does not gate the default views.

## Out of scope

- Verifying that a given entity's commits touch its area — that is the mistag-detection milestone (Tier 2).

## Dependencies

- M-0179 (`paths:` per area) — the oracle this check reads.

## References

- `internal/check/area_unknown.go` — the composition seam (config-sourced declared set) this follows.
