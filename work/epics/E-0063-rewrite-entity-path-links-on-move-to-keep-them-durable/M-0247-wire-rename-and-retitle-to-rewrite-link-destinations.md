---
id: M-0247
title: Wire rename and retitle to rewrite link destinations
status: in_progress
parent: E-0063
depends_on:
    - M-0245
tdd: required
acs:
    - id: AC-1
      title: Rename rewrites entity-body links encoding the old slug to the new slug
      status: open
      tdd_phase: done
    - id: AC-2
      title: Slug-changing retitle rewrites links while a composite-AC retitle rewrites none
      status: open
      tdd_phase: red
---

## Goal

Make `aiwf rename` and slug-changing `aiwf retitle` rewrite the entity-body links
that encode the old slug, so a slug change stops silently rotting cross-links.

## Context

Both verbs change an entity's on-disk slug and rewrite nothing else. `rename`
emits a pure `OpMove` (`internal/verb/rename.go`); `retitle` moves the file and
writes only its own body to sync its `# <id> — title` H1 (`rewriteEntityH1`,
`internal/verb/retitle.go`), never touching other bodies. The mutable part of the
path here is the `<slug>` segment, so the destination transform swaps the slug —
distinct from archive's directory-prefix insert. `retitle` also has a composite-AC
path (retitling an `M-…/AC-…`) that changes no file and must therefore rewrite no
link destinations.

## Acceptance criteria

### AC-1 — Rename rewrites entity-body links encoding the old slug to the new slug

After `aiwf rename <id> <new-slug>`, every entity-body link whose destination
encoded the renamed entity's old slug now carries the new slug and resolves;
unrelated links are unchanged. Evidence: a real-tree integration test — A links to
B by path, rename B, assert A's link resolves.

### AC-2 — Slug-changing retitle rewrites links while a composite-AC retitle rewrites none

A `retitle` that changes the slug rewrites link destinations the same way, in
addition to its existing H1 sync; a composite-AC `retitle` (no file move) rewrites
no link destinations. Evidence: two integration tests, one per branch — the
top-level slug-changing path and the composite-AC no-op path.

## Constraints

- Move-time only; pre-push chokepoint untouched.
- Entity-body writes only.
- `retitle`'s own-H1 sync behavior is preserved, not replaced.

## Design notes

- Destination transform = swap the `<slug>` segment; reuse M-0245 for region
  splitting, resolution, and relative-path recompute.
- The composite-AC path is a genuine no-op for link rewriting and needs a
  traversing test to prove it.
- Decision recorded in `ADR-0033`.

## Surfaces touched

- `internal/verb/rename.go`
- `internal/verb/retitle.go`
- the shared primitive from M-0245

## Out of scope

- `archive` / `reallocate` (sibling milestones).
- Non-entity narrative files.

## Dependencies

- M-0245 — the shared rewrite primitive.

## References

- `internal/verb/rename.go`
- `internal/verb/retitle.go`
- G-0392
