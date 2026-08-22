---
id: M-0314
title: Route move through the shared link-region primitive
status: in_progress
parent: E-0088
tdd: required
acs:
    - id: AC-1
      title: aiwf move routes its link rewriting through the shared primitive
      status: open
      tdd_phase: done
    - id: AC-2
      title: A milestone moved between epics leaves no inbound link broken
      status: open
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
