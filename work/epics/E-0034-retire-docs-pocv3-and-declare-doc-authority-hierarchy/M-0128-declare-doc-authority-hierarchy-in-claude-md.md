---
id: M-0128
title: Declare doc-authority hierarchy in CLAUDE.md
status: draft
parent: E-0034
depends_on:
    - M-0127
tdd: none
acs:
    - id: AC-1
      title: Documentation hierarchy section tags every active docs/ subtree
      status: open
---

## Goal

Add a "Documentation hierarchy" section to CLAUDE.md naming each active `docs/` subtree by authority tier (normative / forward-looking / exploratory / archival). The section is written *once* against the post-M-0127 layout, so the tier labels match the actual path layout a reader will see.

## Context

This is the move G-0092 originally proposed as the minimum (CLAUDE.md table). Writing it before M-0127 would have meant either two passes of the same content or labels that don't match the directory layout; writing it now means once-and-correct.

The mechanical evidence shape — a structural assertion under `internal/policies/` that the named section exists in CLAUDE.md and lists each active `docs/` subtree — is drafted at `aiwfx-start-milestone` time.

## Acceptance criteria

### AC-1 — Documentation hierarchy section tags every active docs/ subtree

CLAUDE.md gains a `## Documentation hierarchy` section naming every currently-active `docs/` subtree and top-level narrative file group, each tagged with exactly one of four closed-set tiers: **normative**, **forward-looking**, **exploratory**, **archival**.

Tier assignment (post-M-0127 layout):

- **Normative** — `docs/adr/`, `docs/design/`, and the top-level operational references (`architecture.md`, `overview.md`, `workflows.md`, `skill-author-guide.md`, `migration/`). Current-truth, kept in lockstep with the code.
- **Forward-looking** — `docs/initiatives/`. Captured ideas awaiting promotion to a real epic/gap entity.
- **Exploratory** — `docs/explorations/` (including `loom/`, `surveys/`), `docs/research/`, `working-paper.md`. Synthesis/thesis/proposal genre; not kernel-binding regardless of internal rigor.
- **Archival** — `docs/archive/` (includes `docs/archive/pocv3/`). Frozen historical snapshot per ADR-0004.

A structural test under `internal/policies/` parses CLAUDE.md's heading hierarchy, locates the `## Documentation hierarchy` section by name, and asserts each subtree name above appears within it tagged with a valid tier label from the closed set. Per the epic's own out-of-scope note, this is a fixed snapshot assertion against the tree as it exists today — not a live drift check against `docs/`'s actual layout (that's G-0092's deferred kernel-rule follow-on).

## Out of scope

- Drift-checking that hierarchy labels match `docs/`'s actual layout at runtime (deferred to G-0092's full kernel-rule follow-on, listed as out of scope in the epic spec).
- Per-tree `_AUTHORITY.md` marker files (option 2 from G-0092's gap body; not the chosen layer per the planning conversation).

## Dependencies

- M-0127 (Relocate) — done. The hierarchy section labels must match the post-Relocate layout.

## References

- **E-0034** — parent epic.
- **G-0092** — superseded by E-0034; this milestone is the concrete realization.
