---
id: M-0251
title: 'Handle #fragment / ?query suffixes in link-destination rewrite'
status: done
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

### AC-2 — Property test: fragment/query preservation holds under generation

Green · commit 800177f1 · tests 2 properties (1500 sampled runs each)

`craftedLink` gained a `suffix` field and the generator gained
`randSuffix`, which appends none, a fragment, a query, or a combined
query-then-fragment suffix (uniformly at random) to each crafted
link's destination. The resolution-correctness oracle
(`TestRewriteLinkDestinations_Property_RewrittenDestinationsResolveToNewPath`)
independently re-splits the rewritten destination on the first `#`/`?`
— a fresh `strings.IndexAny`, not a call into `splitDestinationSuffix`,
so the oracle isn't asserting the primitive against itself — and
checks the extracted suffix equals the generated one exactly, before
resolving the bare path against the move's `To` as before. The
idempotence property needed no code change: suffix-bearing links now
simply flow through the same two-pass check.

Confirmed non-vacuous per `wf-property-test`'s anti-vacuity
discipline: temporarily reverting `splitDestinationSuffix` to a no-op
made the resolution-correctness property fail with a shrunk
counterexample (a suffix-bearing relative link whose bare path no
longer matched the move index); restored before committing. The
independent reviewer separately reproduced this and a second
experiment (dropping just the suffix reattachment), confirming both
halves of the oracle — suffix fidelity and path resolution — are
independently load-bearing.

## Decisions made during implementation

- (none) — all decisions are pre-locked in `## Design notes` above and
  in `ADR-0033`.

## Validation

- `go build ./...` — green.
- `go test -race -parallel 8 ./...` — green, full suite.
- `golangci-lint run` — 0 issues.
- `gofmt -l` on all three touched files — clean.
- `make coverage-gate` — diff-scoped branch-coverage audit and firing-fixture policy tests both pass; every changed line is tested.
- `aiwf check` — 0 error-severity findings (`epic-active-no-drafted-milestones` warning is expected — this is the epic's last drafted milestone; `provenance-untrailered-scope-undefined` is the standard unpushed-branch advisory).
- Independent code-quality review (fresh-context subagent): **approve**, no blocking findings. Independently reran the targeted `TestRewriteLinkDestinations_*` set, the full `internal/verb` race suite, `make coverage-gate`, `make lint`, `gofmt -l`, and `go vet`; reproduced AC-2's non-vacuity claim by re-running the same suffix-splitting revert experiment, and ran a second independent experiment (dropping only the suffix reattachment) that also drove the property red — confirming both the resolution-correctness half and the suffix-fidelity half of the oracle are separately load-bearing; traced the URL-early-return-before-suffix-split ordering, the both-flavor (root-relative and relative) suffix reattachment, and the empty-bare-path same-document-anchor case by hand. Flagged the missing AC-2 Work log entry, since fixed above.
- No design-quality (`wf-rethink`) lens run: this milestone is a narrow addition to one existing function (`rewriteLinkDestination`) plus an extension to an existing property-test generator — no new module boundary, core abstraction, or data model to rethink.
- Doc-lint: skipped — the change-set (`internal/verb/*.go` plus the milestone's own spec) has zero intersection with `docs/`, `README.md`, or `CONTRIBUTING.md`.

## Deferrals

- (none)

## Reviewer notes

The empty-bare-path same-document-anchor case (a destination that is
only a suffix, e.g. `(#some-heading)`, no path at all) is not
explicitly unit-tested. Behavior is safe by construction — the empty
bare path resolves to the linking file's own directory, never matches
an entity's `From` path, so the region is returned byte-identical —
but no test pins this degradation path directly. Low risk, and this
shape isn't a real entity reference in the first place; noted for a
future pass if it ever becomes worth pinning explicitly.

The unit table covers a URL destination with a `#fragment` suffix but
not one with a `?query` suffix. Not a gap in practice — the
`://`-containing early return makes suffix parsing structurally
unreachable for any URL-shaped destination regardless of which
character follows — but flagged as belt-and-suspenders coverage a
future pass could add cheaply.

The property test's independent suffix-splitting oracle
(`linkrewrite_property_test.go`) reimplements the same `#`/`?`
first-occurrence rule as `splitDestinationSuffix`, rather than a
structurally different algorithm. It is a genuinely separate code
path (proven by both revert experiments driving it red), so the
independence holds for implementation bugs, but it would not catch a
shared *conceptual* error in the split rule itself (e.g. if the
correct grammar were actually last-`#` rather than first-`#`/`?`).
Acceptable given the RFC 3986 §4.2 ordering is fixed, documented, and
matches the AC's own stated grammar.

