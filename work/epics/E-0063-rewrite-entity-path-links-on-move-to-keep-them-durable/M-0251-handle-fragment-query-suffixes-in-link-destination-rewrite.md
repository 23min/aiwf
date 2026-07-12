---
id: M-0251
title: 'Handle #fragment / ?query suffixes in link-destination rewrite'
status: in_progress
parent: E-0063
depends_on:
    - M-0245
tdd: required
acs:
    - id: AC-1
      title: 'Preserve #fragment / ?query suffixes on moved-entity rewrite'
      status: met
      tdd_phase: done
    - id: AC-2
      title: 'Property test: fragment/query preservation holds under generation'
      status: met
      tdd_phase: done
---

## Goal

Make the shared link-destination rewrite primitive preserve a `#fragment` or
`?query` suffix when a destination resolves to a moved entity, so an anchored
or query-bearing entity link survives a move exactly like a bare path link
already does.

## Context

`RewriteLinkDestinations` (`internal/verb/linkrewrite.go`, M-0245) treats a
link destination's entire `(...)` contents as a bare path and resolves it via
`path.Clean`. A destination carrying a `#fragment` or `?query` suffix (e.g.
`(docs/adr/ADR-0004-foo.md#uniform-archive)`) never matches a moved entity's
`From` path, so it is left unrewritten — surfaced independently by both review
lenses during M-0245's wrap (G-0409). This milestone closes the gap directly
in the primitive, independent of which wiring milestone (M-0246, M-0247,
M-0248) lands next — each inherits the fix automatically once this milestone
lands, since none of them touch fragment/query parsing themselves.

## Acceptance criteria

### AC-1 — Preserve #fragment / ?query suffixes on moved-entity rewrite

A link destination carrying a `#fragment`, a `?query`, or both (query before
fragment, per the ordering a relative reference uses) has its bare-path
portion split off before resolution; the split path is matched against the
move set exactly as a suffix-free destination is today, and the original
suffix is reattached verbatim on the rewritten destination. A destination
whose bare-path portion does not resolve to a moved entity is left byte-
identical, suffix included. Evidence: a unit table over shapes — fragment-
only, query-only, both combined, crossed with root-relative and relative
flavors and with a matching vs. non-matching move; the untouched-region cases
already pinned by M-0245/AC-1 (URL, code span, fenced block, prose) re-run
with a suffix-bearing destination added to each.

### AC-2 — Property test: fragment/query preservation holds under generation

Extend M-0245/AC-3's generator
(`internal/verb/linkrewrite_property_test.go`) so a crafted link may carry a
randomly-chosen `#fragment`, `?query`, or both, and extend the resolution-
correctness oracle to assert the suffix rides through unchanged while the
bare-path portion resolves to the move's new path — idempotence holds
unchanged. Evidence: `wf-property-test`, same anti-vacuity discipline as
M-0245/AC-3 (confirm the property fails when suffix-stripping is broken,
before declaring done).

## Constraints

- Suffix splitting happens once, before any move-index lookup — no change to
  the existing `walkBodyLines` / `maskCodeSpans` / `splitLinkPathRegions`
  masking primitives.
- A destination whose bare-path portion doesn't match a move is left byte-
  identical, suffix included — same non-mutation guarantee M-0245/AC-1 pins.
- Pure and idempotent, mirroring M-0245's guarantee — the suffix must not be
  re-split or double-processed on a second pass.

## Design notes

- Split the destination on the first `#` or `?` (whichever appears first)
  before calling `resolveLinkDestination`; reassemble by concatenating the
  rewritten bare path with the original suffix. No new masking primitive —
  this is a narrow addition to `rewriteLinkDestination`
  (`internal/verb/linkrewrite.go`), not a new region-splitter.
- Decision recorded in the epic's ADR (`ADR-0033`, *Entity path-links are
  first-class and rewritten on move*) — no new ADR needed; this is scope the
  ADR's invariant already covers, just not yet implemented.

## Surfaces touched

- `internal/verb/linkrewrite.go` — `rewriteLinkDestination`, the function
  this milestone extends
- `internal/verb/linkrewrite_property_test.go` — the generator this milestone
  extends

## Out of scope

- Non-entity `docs/*.md` / README bodies (unchanged epic-wide exclusion).
- Wiring into any verb — archive/rename/retitle/reallocate wire into the
  primitive independently in their own milestones; this milestone only
  hardens the primitive itself.

## Dependencies

- M-0245 — the primitive this milestone extends.

## References

- G-0409 — the gap this milestone closes
- `internal/verb/linkrewrite.go`
- `internal/verb/linkrewrite_property_test.go`

---

## Work log

### AC-1 — Preserve #fragment / ?query suffixes on moved-entity rewrite

Green · commit 1d5b1a0d · tests 2 funcs / 11 subtests

`rewriteLinkDestination` (`internal/verb/linkrewrite.go`) now splits a
`#fragment`/`?query` suffix off the destination via a new
`splitDestinationSuffix` — the first `#` or `?` in the string marks
the suffix's start, matching a relative reference's query-before-
fragment ordering (RFC 3986 §4.2), so a combined `?query#fragment` is
carried as one verbatim block. The bare path is resolved and matched
against the move index exactly as before; a rewrite reattaches the
suffix verbatim, and a non-matching bare path leaves the whole
destination, suffix included, byte-identical — same non-mutation
guarantee as M-0245/AC-1.

Two new test functions: a 7-case table covering fragment-only,
query-only, and combined suffixes crossed with root-relative and
relative flavors and matching vs. non-matching moves; and a 4-case
re-run of M-0245/AC-1's untouched-region cases (URL, code span,
fenced block, prose) with a suffix added to each, confirming suffix
support doesn't leak past the existing masking boundaries. All
pre-existing `linkrewrite*_test.go` tests — including both M-0245/AC-3
property tests — pass unmodified, confirming this is additive, not a
behavior change to the suffix-free path.

