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
      status: met
      tdd_phase: done
    - id: AC-2
      title: Slug-changing retitle rewrites links while a composite-AC retitle rewrites none
      status: met
      tdd_phase: done
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

---

## Work log

### AC-1 — Rename rewrites entity-body links encoding the old slug to the new slug

Green · commits f2d3d283, a4e14007 · tests 3/3

Added `renameEntityMoves` (`internal/verb/rename.go`) — the same
`pathInside`/`newEntityPathAfterRename` directory-expansion pattern
`archiveEntityMoves` uses for M-0246 — plus a new shared
`planLinkRewriteWrites` (`internal/verb/linkrewrite_ops.go`) that
walks every active entity and emits an `OpWrite` for any body whose
link resolves to a moved path. `Rename` now appends the computed
rewrite ops after its own `OpMove`. The diff-scoped coverage gate
caught an unreachable empty-moves guard (both call sites already
prevent it) and an untested sort comparator; `a4e14007` drops the
dead code and adds the missing coverage. Three real-tree tests: a
slug-swap producing two rewrites in one call (exercising the sort),
a directory-shaped epic rename whose own body links to a co-moved
nested milestone, and an already-archived-entity exclusion test
mirroring M-0246's identical rule for archive.

### AC-2 — Slug-changing retitle rewrites links while a composite-AC retitle rewrites none

Green · commit 916df6c5 · tests 3/3

A slug-changing retitle computes the same move set via
`renameEntityMoves` and folds `RewriteLinkDestinations` into the body
it already rewrites for the H1 sync, before serializing — so the two
concerns land in one write to `contentPath`, not two competing
`OpWrite`s for the same path. `planLinkRewriteWrites` then covers
every other entity, with the retitled entity's own (pre-move) path
excluded since it was already handled explicitly. Three real-tree
tests: the slug-changing path composing H1 sync with a link rewrite on
an unrelated linking entity, a directory-shaped epic retitle whose own
body links to a co-moved nested milestone (proving the single-write
composition), and the composite-AC path asserting its `Plan` carries
exactly its one pre-existing write and nothing else.
