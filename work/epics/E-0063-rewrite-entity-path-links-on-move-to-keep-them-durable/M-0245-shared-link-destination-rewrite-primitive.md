---
id: M-0245
title: Shared link-destination rewrite primitive
status: draft
parent: E-0063
tdd: required
acs:
    - id: AC-1
      title: Rewrite link destinations to a moved entity, leaving prose and code untouched
      status: open
      tdd_phase: red
    - id: AC-2
      title: Recompute relative link destinations against the linking file directory
      status: open
      tdd_phase: red
    - id: AC-3
      title: Rewrite core is idempotent; rewritten destinations resolve to new paths
      status: open
      tdd_phase: red
---

## Goal

Build the pure, idempotent primitive that rewrites markdown link destinations in
an entity body given a set of entity moves — the shared engine every file-moving
verb in this epic wires into.

## Context

`rewidth` already rewrites link destinations the safe way: `linkPathPattern` plus
the `splitLinkPathRegions` / `rewriteOutsideChunk` region-splitter operate only on
`](…)` destination tokens, exclude code fences / inline code / URLs, leave
`/archive/` and external paths alone, and are pure and idempotent. That machinery
is root-relative (`(work/…)`) and width-only, so it never handled the relative
`../…/work/…` links that actually rot. This milestone lifts it into a shared
`internal/verb` primitive and generalizes it to relative destinations. Nothing
wires it into a verb yet — that lands in the sibling milestones — so this ships as
tested, unused-by-verbs library code.

## Acceptance criteria

### AC-1 — Rewrite link destinations to a moved entity, leaving prose and code untouched

Given a move set (old path → new path for each moved entity) and an entity body,
the primitive rewrites every markdown link whose destination resolves to a moved
entity and leaves everything else byte-identical: prose, inline-code spans, fenced
code blocks, URL-shaped tokens, and links whose destination is not a moved entity.
Evidence: a unit table over fixtures covering each preserved region class and the
rewrite case; the non-match and preserved-region arms each have a traversing test.

### AC-2 — Recompute relative link destinations against the linking file directory

A relative destination (`](../work/…)`, `](../../work/…)`, any `../` depth) is
recomputed against the linking file's own directory so the rewritten link resolves
to the target's new location; root-relative destinations keep working. Evidence: a
golden fixture reproducing the rot shape seen in ADR bodies (a sibling-directory
link into `work/`) with synthetic ids — the link resolves after the rewrite.

### AC-3 — Rewrite core is idempotent; rewritten destinations resolve to new paths

Running the primitive twice on a body yields the same output as running it once,
and every rewritten destination resolves to the moved entity's new path. Evidence:
a `wf-property-test` generating tree layouts and move sets, asserting both
properties across all generated cases.

## Constraints

- Pure and idempotent — no I/O in the rewrite core, mirroring `rewidth`'s
  guarantee.
- Destination-token-scoped: only `](…)` regions are touched; prose, code, and
  URLs are masked exactly as `rewidth` masks them today.
- Path handling uses the repo's `filepath.ToSlash` slash discipline.

## Design notes

- Reuse, don't reinvent: the extraction lifts `rewidth`'s existing region-splitter
  and predicate structure into a shared location both `rewidth` and the new
  callers use.
- Decision recorded in the epic's ADR (`ADR-NNNN`, *Entity path-links are
  first-class and rewritten on move*).

## Surfaces touched

- `internal/verb/rewidth.go` — source of the machinery being lifted
- `internal/verb/` — new shared primitive

## Out of scope

- Wiring into any verb (archive / rename / retitle / reallocate land in the
  sibling milestones).
- Non-entity `docs/*.md` / `README` bodies.

## Dependencies

- None — this is the foundational milestone the others depend on.

## References

- `internal/verb/rewidth.go`
- G-0392 — the gap this epic addresses
