---
id: ADR-0033
title: Entity path-links are first-class and rewritten on move
status: accepted
---
# ADR-0033 — Entity path-links are first-class and rewritten on move

> **Date:** 2026-07-09 · **Decided by:** Peter Bruinsma

## Context

aiwf references an entity two ways. A bare id is id-addressed: the loader resolves
it across the active tree and `archive/` by construction, so it survives any move.
A markdown path-link is path-addressed: it bakes a static relative filesystem path
into prose. The file-moving verbs — `archive` (subtree `git mv` into `archive/`),
`rename` and `retitle` (slug change), `reallocate` (id change) — rewrite little or
nothing in other entities' bodies, so bare-id references survive while path-links
rot. Measured directly: three of four `docs/adr` files linking into `work/` were
broken by since-moved targets, a 75% rot rate in the most actively-maintained
corner of the docs tree.

Two responses were on the table. The first was to ban path-links — a new pre-push
check steering authors to bare-id citation. It is rejected here: a bare id is not
clickable in GitHub or an editor, so the ban taxes a real authoring convenience,
and there is no source-markdown form that is both clickable and archive-proof. The
second — keep path-links and repair them when their target moves — is chosen. The
machinery already exists: `rewidth` rewrites link destinations the safe way
(destination-token-scoped; code, inline-code, URL, and archive paths excluded;
pure and idempotent), limited only to root-relative links and the id-width
transform.

## Decision

Entity path-links are a first-class reference form. We do not ban them and do not
require bare-id citation for navigation.

- Every verb that changes an entity's on-disk path rewrites the markdown link
  destinations in entity bodies that point at it — relative or root-relative —
  through one shared link-region primitive generalized from `rewidth`'s machinery.
  Prose, inline code, fenced code, URLs, and external paths are left untouched.
- The primitive rewrites only files the loader owns (entity bodies). Non-entity
  narrative (`README`, `CONTRIBUTING`, non-entity `docs` files) is covered by the
  advisory `wf-doc-lint` markdown-link-integrity check (G-0390), not auto-rewritten
  — a verb commit must not reach outside the entity set it owns.
- Enforcement is at move-time only. No pre-push check rule is added for this
  concern, so the pre-push chokepoint's cost is unchanged.
- ADR-0004's move-based archive is preserved. Archiving still physically moves the
  file; no redirect stub or tombstone is introduced.
- Bare-id citation remains the form for running-prose mentions, where a link would
  be noise.

## Consequences

- Path-links between entity bodies stay correct across `archive`, `rename`,
  `retitle`, and `reallocate` without author vigilance; the measured rot class is
  closed at its source.
- `rewidth`'s link-rewrite machinery is lifted into a shared primitive and
  generalized to relative destinations — the form that actually rotted, which
  `rewidth`'s root-relative pattern never covered. `reallocate`'s incidental
  id-substring path rewriting is unified onto the primitive for link-region
  precision.
- `archive` and `rename` commits grow: they now also write the entity bodies whose
  links point at a moved entity, widening the commit's blast radius and merge
  surface. `reallocate` already accepts this trade.
- The residual is deliberate: links from non-entity files, and links broken by a
  raw `git mv` that bypasses the verbs, are covered by advisory detection only, not
  the mechanical guarantee.

## Validation

A golden fixture reproducing the rot shape (a sibling-directory link into `work/`)
stays green: the link resolves after its target is archived, renamed, or retitled.
If the shared primitive's cost ever appears on the pre-push path, the move-time-only
property has been violated and the decision needs a revisit.

## References

- Linked epic: E-0063
- G-0392 — the gap this decision addresses
- G-0390 — the advisory `wf-doc-lint` markdown-link-integrity backstop
- ADR-0004 — uniform archive convention (preserved)
