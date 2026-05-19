---
id: M-0128
title: Declare doc-authority hierarchy in CLAUDE.md
status: draft
parent: E-0034
depends_on:
    - M-0131
tdd: none
---

## Goal

Add a "Documentation hierarchy" section to CLAUDE.md naming each active `docs/` subtree by authority tier (normative / forward-looking / exploratory / archival). The section is written *once* against the post-M-0131 layout, so the tier labels match the actual path layout a reader will see.

## Context

This is the move G-0092 originally proposed as the minimum (CLAUDE.md table). Writing it before M-0131 would have meant either two passes of the same content or labels that don't match the directory layout; writing it now means once-and-correct.

The mechanical evidence shape — a structural assertion under `internal/policies/` that the named section exists in CLAUDE.md and lists each active `docs/` subtree — is drafted at `aiwfx-start-milestone` time.

## Out of scope

- Drift-checking that hierarchy labels match `docs/`'s actual layout at runtime (deferred to G-0092's full kernel-rule follow-on, listed as out of scope in the epic spec).
- Per-tree `_AUTHORITY.md` marker files (option 2 from G-0092's gap body; not the chosen layer per the planning conversation).

## Dependencies

- M-0131 (Relocate) — done. The hierarchy section labels must match the post-Relocate layout.

## References

- **E-0034** — parent epic.
- **G-0092** — superseded by E-0034; this milestone is the concrete realization.
