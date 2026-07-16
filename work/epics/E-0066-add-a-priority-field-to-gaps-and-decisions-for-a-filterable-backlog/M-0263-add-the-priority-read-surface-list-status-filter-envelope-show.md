---
id: M-0263
title: 'Add the priority read surface: list/status filter, envelope, show'
status: draft
parent: E-0066
depends_on:
    - M-0261
tdd: required
---

# M-0263 — Add the priority read surface: list/status filter, envelope, show

## Goal

Make `priority` queryable and visible on the text and JSON surfaces: a `--priority <level>` filter on `aiwf list` and `aiwf status`, the value on the JSON envelope entity payload, and `aiwf show` surfacing it.

## Context

Once the field exists (field milestone) and can be set (write-surface milestone), the backlog's original friction — "picking which one to work next requires reading every body" — is answered by filtering. This milestone adds the read paths. Ordering (group-by-status, priority-as-tiebreaker) is explicitly not here; it's deferred to G-0420, so this ships filtering only over the existing id-order sort.

## Acceptance criteria

<!-- Seeded via `aiwf add ac`; each starts at tdd_phase: red. -->

## Constraints

- `--priority` filters the result set only; it does not change sort order (that is G-0420). The existing id-order sort is untouched.
- The filter is a closed-set value validated the same way the writers validate it — a bad `--priority` value is a usage error, not a silent empty result.
- The JSON envelope carries `priority` on the entity payload through the existing serialization boundary, not a bespoke side-channel.

## Design notes

- Confirm whether a render/list contract pins the entity JSON payload shape before adding the field to the envelope — if so, the contract needs a coordinated bump (open question carried from the epic).
- `aiwf show` may surface the field incidentally if it renders all frontmatter, or need a one-line addition — determine at implementation.

## Surfaces touched

- `internal/cli/list/`, `internal/cli/status/` — the `--priority` filter flag and predicate.
- The JSON envelope entity payload; `aiwf show`.

## Out of scope

- Sort ordering by priority — G-0420.
- The HTML badge (the render milestone).

## Dependencies

- M-0261 — the field and closed-set predicate must exist first. Independent of the write-surface milestone (test fixtures set the field directly).

## References

- G-0078 — the ratified design decisions (filter-only for v1).
- G-0420 — the deferred sort-ordering follow-up.
