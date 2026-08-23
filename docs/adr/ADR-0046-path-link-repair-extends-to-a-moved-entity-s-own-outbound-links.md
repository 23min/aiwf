---
id: ADR-0046
title: Path-link repair extends to a moved entity's own outbound links
status: proposed
---
# ADR-0046 — Path-link repair extends to a moved entity's own outbound links

> **Date:** 2026-08-23 · **Decided by:** Peter Bruinsma

## Context

[ADR-0033](ADR-0033-entity-path-links-are-first-class-and-rewritten-on-move.md)
commits every path-changing verb to repairing markdown links "in entity bodies
that point at it" — the inbound direction. A moved file's *own* links were left
where they were, and they break for the mirror-image reason: a relative
destination is resolved against the directory the file sits in, so relocating
the file changes what its unchanged text means.

Measured 2026-08-19, sweeping ADR-0003 into `docs/adr/archive/`: five of its
outbound links broke, along with two inbound links held by an already-archived
sibling. The `link-check` workflow reported them; no verb did. They were
repaired by hand.

The direction is the only thing that differs. The rot class is the same — a
path-link between entity bodies going stale because a verb moved a file — and
the repair is the same primitive, with the same discrimination between prose,
inline code, fenced code, URLs and external paths.

ADR-0033's inbound-only wording tracks the evidence that motivated it. Its
measurement was three of four `docs/adr` files linking into `work/`, all
inbound; nothing in it weighs the outbound direction and rejects it.

## Decision

ADR-0033's commitment covers a moved entity's own outbound links. This extends
that decision rather than replacing it: ADR-0033 stays accepted and remains the
operative record for everything else it decided.

- A verb that changes an entity's on-disk path rewrites the markdown link
  destinations **in that entity's own body** as well as in the bodies pointing
  at it, so a relative destination keeps naming the same file after the move.
- The boundary is unchanged. ADR-0033's second bullet limits the primitive to
  files the loader owns, and a moved entity's own body is such a file. Nothing
  here reaches into non-entity narrative, which stays covered by the advisory
  `wf-doc-lint` link-integrity check.
- The existing discrimination holds in both directions. Prose, inline code,
  fenced code, URLs and external paths stay untouched outbound exactly as they
  do inbound.
- Enforcement stays at move time. No pre-push check rule is added, so ADR-0033's
  unchanged-chokepoint-cost property is preserved.

## Consequences

- The rot class ADR-0033 set out to close at its source is now closed in both
  directions. Archiving a file no longer breaks the links it carries, which is
  the shape the 2026-08-19 observation recorded.
- A mover now edits the bytes of the file it is relocating, which the movers
  previously did only incidentally. That composition is safe: a plan may carry
  an `OpMove` of a file and an `OpWrite` at that file's new path, and a failure
  after both land restores the worktree fully-old, because the rollback journal
  replays in strict LIFO order. Reversing that order strands a duplicate at the
  destination and leaves edited bytes at the origin.
- Commits grow again, by the moved entity's own body. This is the smaller
  increment: that file is already in the commit, so the blast radius gains
  content but no new path.
- A moved file whose outbound link was *already* broken before the move is not
  repaired. The primitive rewrites destinations that resolve to a moved entity;
  it does not validate the ones that do not.

## Validation

A moved entity carrying relative links to siblings keeps every one of them
resolving after the move, asserted by resolving the destinations on disk rather
than pattern-matching the rewritten text. The inbound fixtures ADR-0033 named
stay green unchanged — extending the reach must not alter what it already
guarantees.

## References

- ADR-0033 — the decision this extends; inbound-only as written, and still the
  operative record for the rest of what it decided
- ADR-0004 — the move-based archive convention whose sweep exposed the outbound
  half
- E-0088 — the epic that measured the divergence and carries the work
