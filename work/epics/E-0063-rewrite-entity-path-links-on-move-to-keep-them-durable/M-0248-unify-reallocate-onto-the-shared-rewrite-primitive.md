---
id: M-0248
title: Unify reallocate onto the shared rewrite primitive
status: in_progress
parent: E-0063
depends_on:
    - M-0245
tdd: required
acs:
    - id: AC-1
      title: Reallocate rewrites path-links via the shared primitive, not prose id tokens
      status: met
      tdd_phase: done
---

## Goal

Route `reallocate`'s path-link rewriting through the shared primitive so it is
link-region-scoped and precise, while keeping its bare-id prose rewrite for
non-link mentions.

## Context

`reallocate` already rewrites references, but via an id-token substring replace
(`idPattern.ReplaceAll`, `internal/verb/reallocate.go`) that is not link-region
aware. It lands the right path only incidentally — the slug is unchanged, so
swapping the id substring inside a path happens to produce the correct filename —
and the same substring pass can touch an id-shaped token in prose or a code span.
This milestone is a refinement, not a rot fix: `reallocate` works today. It exists
so the epic leaves one consistent link-rewrite path rather than two mechanisms.
Optional — droppable if the epic tightens.

## Acceptance criteria

### AC-1 — Reallocate rewrites path-links via the shared primitive, not prose id tokens

`reallocate`'s path-link rewriting goes through M-0245's primitive: a real markdown
link to the old id is rewritten to the new id's path, while an old-id token inside
a code span or plain prose is handled only by the separate bare-id prose pass and
is not additionally rewritten by the link path. Evidence: a unit test asserting the
link-vs-prose precision boundary — a fixture where the same old id appears both in
a link destination and in a code span, with only the link destination rewritten by
the primitive.

## Constraints

- The bare-id prose rewrite (non-link mentions) is preserved.
- No behavior change to the ids `reallocate` produces — this changes *how* path
  links are rewritten, not *what* the new id is.

## Design notes

- The link path moves to M-0245's region-splitter; the id-token prose pass stays
  for bare mentions. The two no longer overlap on link destinations.
- Decision recorded in `ADR-0033`.

## Surfaces touched

- `internal/verb/reallocate.go`
- the shared primitive from M-0245

## Out of scope

- `archive` / `rename` / `retitle` (sibling milestones).
- Any change to id allocation semantics.

## Dependencies

- M-0245 — the shared rewrite primitive.

## References

- `internal/verb/reallocate.go`
- G-0392

---

## Work log

### AC-1 — Reallocate rewrites path-links via the shared primitive, not prose id tokens

Green · commit 64cde8a6 · tests 12/12

`reallocate` now composes two non-overlapping passes per touched body:
M-0245's `RewriteLinkDestinations` rewrites a real markdown link to
the renumbered entity's old path first, then a new
`rewriteBareIDMentions` (`internal/verb/reallocate.go`) rewrites every
remaining bare id-token mention — prose, a link's own visible text, a
code-span mention — while explicitly excluding link-path destination
regions (reusing `splitLinkPathRegions` from M-0245's region-splitter
so both passes agree on what counts as "inside a link destination").
`renameEntityMoves` (M-0247) supplies the `EntityMove` set, reused
as-is rather than duplicated a third time.

The red test proves a genuine behavior change, not just an
architecture-only refactor: the prior blind `idPattern.ReplaceAll`
corrupted a URL-shaped link destination that merely contained the old
id as a substring (e.g. `https://example.com/issues/G-0001`), since
it could not distinguish a real entity-path reference from an
unrelated id-shaped token. The region-aware primitive leaves it
byte-identical. All 12 `TestReallocate_*` tests green, including the
new fixture asserting the link/URL/code-span precision boundary in
one body.
