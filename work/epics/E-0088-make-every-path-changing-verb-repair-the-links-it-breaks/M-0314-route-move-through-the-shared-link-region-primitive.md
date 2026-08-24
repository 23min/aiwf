---
id: M-0314
title: Route move through the shared link-region primitive
status: done
parent: E-0088
tdd: required
acs:
    - id: AC-1
      title: aiwf move routes its link rewriting through the shared primitive
      status: met
      tdd_phase: done
    - id: AC-2
      title: A milestone moved between epics leaves no inbound link broken
      status: met
      tdd_phase: done
---

## Goal

Close the one place where ADR-0033's first bullet is unmet: `aiwf move` changes
an entity's on-disk path and rewrites nothing.

## Context

Five verbs emit an `OpMove`. Four route through the shared link-region
primitive — `archive`, `reallocate`, `rename`, `retitle`. `move` computes its
destination from the target epic's directory and calls neither
`planLinkRewriteWrites` nor `RewriteLinkDestinations`. The entity-truth audit
records this as its only `contradicted-by-code` verdict against ADR-0033, and
the test suite carries `archive_`, `rename_`, `retitle_` and `reallocate_`
link-rewrite tests with no `move` counterpart.

The primitive already exists and is exercised by four callers, so this milestone
adds a call site rather than a mechanism.

## Acceptance criteria

### AC-1 — aiwf move routes its link rewriting through the shared primitive

`aiwf move` plans link-rewrite writes through the same primitive its four
sibling movers use. Evidence is a test that fails if the call is removed — not a
grep for the identifier, which would pass against a call that never executes on
the move path.

### AC-2 — A milestone moved between epics leaves no inbound link broken

End to end in a disposable tree: an entity body links to a milestone by path;
the milestone moves to another epic; the link still resolves. The assertion
reads the rewritten bytes, not the plan.

## Constraints

- **Route through the existing primitive.** A second implementation beside it is
  the failure mode this milestone exists to avoid.
- **Inbound only.** Links pointing *at* the moved entity are ADR-0033's
  commitment; the moved file's own outbound links are the next milestone's
  subject and are out of scope here.
- **No behavior change to the other four movers.** Their existing link-rewrite
  tests stay green untouched.

## Design notes

ADR-0033 is the specification. The primitive's discrimination between prose,
inline code, fenced code, URLs and external paths is established behavior to
reuse, not to re-derive.

## Out of scope

- Outbound link rewriting.
- Any link outside the entity set the verb owns — ADR-0033's second bullet.
- New check rules; ADR-0033's third bullet places enforcement at move time.

## Dependencies

None. The primitive exists.

## References

- ADR-0033 — the specification
- E-0088 — the parent epic
- `internal/verb/move.go`, `linkrewrite.go`, `linkregion.go`

## Work log

### AC-1 — aiwf move routes its link rewriting through the shared primitive

`move` plans inbound link repair through `planLinkRewriteWrites`, excluding its
own file · commit 051daad1a · internal/verb 634 passed, 1 skipped, 0 failed

The exclude set is load-bearing rather than defensive, and only in one shape: it
matters when the moved file's own body carries a link resolving into the move
set. There the helper emits a second write for the destination path, serialized
from the *tree's* entity — which still holds the pre-move `parent:` — competing
with the write `move` already plans to update that field. Measured under
mutation: dropping the exclude produces two writes at one path, one naming the
destination epic and one still naming the source epic. A fixture whose milestone
body links to nothing that moves cannot discriminate the two implementations at
all, so the self-link is what gives that assertion its power.

The moved file's own outbound self-link is left pointing at the pre-move path.
That is M-0315's subject, so nothing here asserts its post-move destination.

### AC-2 — A milestone moved between epics leaves no inbound link broken

An end-to-end sweep resolves every inbound link on disk after the move and stats
its destination · commit b967c3352 · internal/verb 635 passed, 1 skipped, 0 failed

Resolving rather than string-comparing is what makes this an integrity claim: a
link rewritten to a path nothing occupies satisfies a `Contains` assertion and
fails a `stat`. The sweep covers both destination flavors and skips the moved
file, whose own self-link stays pointing at the pre-move path until M-0315.

Removing the `://` guard in `rewriteLinkDestination` leaves these tests green,
but the mutant is **not** equivalent, and M-0316 should record it as a live
survivor rather than an excused one. The guard tests the whole destination
before `splitDestinationSuffix` separates a `?query` / `#fragment` suffix, so a
destination whose suffix carries `://` is distinguished:
`…/M-NNNN-<slug>.md?u=https://example.com` is left alone with the guard and
rewritten without it. A scheme in *scheme position* is what the guard means to
catch, and testing the whole string reaches further than that.

The same measurement is a live ADR-0033 hole rather than only a test gap: those
destinations name the moved entity and no mover rewrites them. Tracked as
G-0622, which owns the fix; this milestone neither causes it nor fixes it.

## Decisions made during implementation

None — all decisions are pre-locked above. The one choice with alternatives, that
`move` excludes its own file from the shared helper rather than folding an
inline rewrite into its existing write, is settled by the *Inbound only*
constraint: the inline form is outbound rewriting, which is M-0315's subject.

## Validation

Measured on the reconciled milestone branch, after the review corrections:

- `make check-fast` — exit 0. `go vet` across the untagged, `stress` and
  `testpins` tag sets; `go test -race -parallel 8 ./...` reporting `ok` for all
  71 packages with no `FAIL`; `golangci-lint run` clean.
- `AIWF_COVERAGE_BASE=epic/E-0088-make-every-path-changing-verb-repair-the-links-it-breaks make coverage-gate`
  — passes. No changed line is uncovered, and the firing-fixture meta-gate holds.
- `go test ./internal/verb/` — 635 passed, 1 skipped, 0 failed. The skip predates
  this milestone.
- `go build ./...` — exit 0.
- `aiwf check` — 0 errors, 7 warnings, every one inherited from mainline
  (`terminal-entity-not-archived` for five swept-pending gaps,
  `archive-sweep-pending`, and `provenance-untrailered-scope-undefined`, which
  reports only that an unpushed branch has no upstream to audit against).

## Deferrals

- `G-0622` — the shared primitive's URL guard tests the whole destination before
  the `?query` / `#fragment` suffix is split off, so an entity link whose suffix
  carries `://` is never rewritten and breaks on a move. Found while auditing
  this milestone's own assertions; pre-existing, in code this milestone does not
  change, and the fix is a behaviour change to a shipped surface needing its own
  decision. Every mover routed through the primitive carries it, not just `move`.

## Reviewer notes

An independent fresh-context reviewer read the full change-set and returned
approve with no blocking findings. It confirmed by mutation that dropping the
exclude set leaves the milestone filed under one epic while its frontmatter
names another — a worse failure than the competing-write framing used above —
and that AC-1's test is the only thing in the tree that catches it. The
`//coverage:ignore` was checked by stripping the directive and re-running the
gate: it exempts exactly the line it names.

No design-quality pass was run, deliberately. `wf-rethink` is per-unit by rule
and this milestone introduced no unit for it: one call site added to an existing
verb, routed through an existing helper, with no new boundary, abstraction or
data model.

Fixed in place rather than deferred, both cheap and in files this milestone
already owned: the AC-2 test's resolver matched `work/` and `docs/` where the
production rule matches five specific entity directories, so a destination under
a non-entity directory would have been checked at a path the primitive never
produces; and `aiwf move --help` documented none of the link repair this
milestone added, against the repo's rule that discoverability ships with the
implementation. The help text now also names the refusal a reader would
otherwise meet unexplained — an uncommitted edit to any entity linking at the
moved milestone now blocks the move, because those bodies joined the verb's
write set.

Considered and declined, so the next reviewer meets a decision rather than a
blank:

- Routing `moves` through `renameEntityMoves` instead of building the single
  `EntityMove` directly. Its only arm reachable from `move` is the `default:`
  one, which returns the same literal; the directory-expansion arm cannot be
  reached, since `move` refuses non-milestones. Indirection through a
  rename-named helper would cost a misleading name and buy nothing.
- Asserting the moved file's own self-link after the move. It stays pointing at
  the pre-move path, and pinning that would pin behaviour M-0315 exists to
  change.
- The URL assertion in AC-1 does not kill the `://`-guard mutant, and is kept
  anyway: it discriminates a different mutant class, and the property it states
  is one ADR-0033 names.
