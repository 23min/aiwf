---
id: M-0314
title: Route move through the shared link-region primitive
status: in_progress
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
own file · commit 051daad1a · tests 635/635 green (+1 test)

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
its destination · commit b967c3352 · tests 636/636 green (+1 test)

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
