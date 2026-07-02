---
id: M-0222
title: Path-scoped single-entity history with bloom-filter maintenance
status: cancelled
parent: E-0054
tdd: required
acs:
    - id: AC-1
      title: single-entity history resolves via path-scoped git log over the entity path set
      status: deferred
      tdd_phase: red
    - id: AC-2
      title: aiwf update maintains changed-path bloom filters idempotently
      status: deferred
      tdd_phase: red
    - id: AC-3
      title: path-scoped history equals trailer-grep history including renamed entities
      status: deferred
      tdd_phase: red
    - id: AC-4
      title: measured single-entity history wall-time delta recorded in Validation
      status: deferred
      tdd_phase: red
---
## Goal

**Cancelled during the E-0054 pre-start review — superseded by G-0340.**

This milestone's original premise was that a path-scoped `git log -- <path>` can
replace the trailer grep for single-entity history, with the path set reconstructable
from current path + `prior_ids` + the archive convention. The review showed that
premise is false:

- `prior_ids` tracks only `reallocate` id-lineage — not `rename` slug changes,
  `archive` moves, or transitive parent-dir moves — so the historical path set is
  **not** reconstructable from frontmatter.
- Pathless `--allow-empty` commits (`acknowledge`, `authorize`, `audit-only`) carry
  `aiwf-entity:` but touch no file, so a path query cannot see them.
- `git log -- <path>` applies history simplification, pruning merge commits the
  trailer grep retains.

The deferred work — path-scoped history as a *verified accelerator* over the
trailer-grep oracle, plus changed-path bloom-filter maintenance — is tracked in
**G-0340**, which records these constraints. The safe read-verb win extracted from
this milestone (guard the unconditional authorize grep) landed as **M-0223**.

## Notes

Do not resurrect this body's original path-set assumption; see G-0340 for the correct
framing (trailer grep stays the authoritative oracle).

### AC-1 — single-entity history resolves via path-scoped git log over the entity path set

Deferred to G-0340.

### AC-2 — aiwf update maintains changed-path bloom filters idempotently

Deferred to G-0340.

### AC-3 — path-scoped history equals trailer-grep history including renamed entities

Deferred to G-0340.

### AC-4 — measured single-entity history wall-time delta recorded in Validation

Deferred to G-0340.
