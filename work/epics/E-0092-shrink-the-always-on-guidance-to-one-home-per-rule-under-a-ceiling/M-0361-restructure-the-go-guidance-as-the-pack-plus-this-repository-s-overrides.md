---
id: M-0361
title: Restructure the Go guidance as the pack plus this repository's overrides
status: draft
parent: E-0092
depends_on:
    - M-0338
tdd: none
---
## Goal

Make Go reach its language pack the way the other languages do: `.guidance/project/go-conventions.md` holds only what this repository adds to or overrides in the Go pack, the router sends Go work to both, and a before-and-after observation shows whether Claude now reads the pack.

## Closes

- (none)

## Context

M-0338's after-observation found Go the one language with a project document on the router, an entry that presents `go-conventions.md` as overriding the pack. `go-conventions.md` repeats part of the pack, and Claude took it in the pack's place: it read the Go pack in no Go run of the round and in one of five T3 runs with the router's pack route, while Codex read both. Making `go-conventions.md` primed was ruled out, since it takes the primed load over M-0349's targets and makes Go the one language always loaded. M-0338's Validation and Decisions made during implementation hold the measurements.

## Acceptance criteria

<!-- Planned with aiwfx-plan-milestones after M-0338's wrap. -->

## Constraints

- Removing text from `go-conventions.md` changes M-0349 row decisions, so each change is a maintainer decision recorded in M-0349, and the inventory test's reconciliation must accept the blocks that carried a row's earlier decision.
- The observation reuses `scripts/e0092-observe/` at its pin and M-0334's rubric as committed, with a baseline at M-0338's final commit for T1, T3 and T11 on both hosts.
- `guidance.enabled` stays `false` until this milestone completes.

## Design notes

- (planned with the acceptance criteria)

## Out of scope

- Changing the Go pack itself; its wording belongs to the external source.
