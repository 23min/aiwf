---
id: M-0315
title: Rewrite a moved entity's own outbound links
status: in_progress
parent: E-0088
depends_on:
    - M-0314
tdd: required
acs:
    - id: AC-1
      title: The write path is shown safe or unsafe for editing the file being moved
      status: open
      tdd_phase: done
    - id: AC-2
      title: A decision records whether ADR-0033 reaches outbound links
      status: open
    - id: AC-3
      title: A moved entity's own relative links resolve after the move
      status: open
---

## Goal

Make a file that changes directory keep its own relative links resolving, and
record the decision that extends ADR-0033's reach to cover it.

## Context

ADR-0033 commits the primitive to rewriting links "in entity bodies that point
at it" — inbound only. A moved file's own links were never in scope, so when a
file moves into an `archive/` subdirectory its bare-filename links resolve
against the new directory and break.

Observed 2026-08-19: sweeping ADR-0003 into `docs/adr/archive/` broke five of
its outbound links, and two more inbound links held by an already-archived
sibling. The `link-check` workflow reported them; no verb did. They were
repaired by hand.

This milestone is the one place in E-0088 that knowingly reaches past the
ratified specification, which is why the decision lands before the code.

## Acceptance criteria

### AC-1 — The write path is shown safe or unsafe for editing the file being moved

The movers today do not edit the content of the file they relocate; they move it
and rewrite *other* files. Whether the atomic-write path can carry a
content-edit-plus-move for the same file is answered by demonstration — a test
that exercises the combined operation and asserts the on-disk result is
fully-old or fully-new, never half-written. A negative answer is a valid
outcome and reshapes AC-3.

### AC-2 — A decision records whether ADR-0033 reaches outbound links

A decision record settles whether outbound rewriting is an extension of ADR-0033
or a separate commitment, and says which. Evidence is the record existing and
being reachable from ADR-0033 — not prose in this milestone asserting the
question was considered.

### AC-3 — A moved entity's own relative links resolve after the move

End to end in a disposable tree: an entity carrying relative links to siblings
moves into an `archive/` subdirectory; every one of its outbound links still
resolves. The assertion resolves the link targets on disk rather than pattern-
matching the rewritten text.

## Constraints

- **The decision precedes the code.** AC-2 is not paperwork filed after the fact.
- **Same primitive.** Outbound rewriting extends the existing link-region
  machinery; it does not fork it.
- **Prose, inline code, fenced code, URLs and external paths stay untouched** —
  the existing discrimination holds for outbound as it does for inbound.
- **Still inside the owned entity set.** Rewriting a moved entity's own body is
  within what the verb owns; this milestone does not reach into `docs/`.

## Design notes

The observed failure is the shape to test against: a file whose links were
written as bare filenames valid in its original directory, moved one level
deeper. Root-relative and `../`-prefixed forms behave differently under the same
move and both need cases.

## Out of scope

- Links in `docs/` — ADR-0033's second bullet, and M-0317's subject.
- Redirect stubs or tombstones at the vacated path — ADR-0033's fourth bullet.

## Dependencies

- M-0314 — establishes that every mover routes through the primitive, which is
  the seam this milestone extends.
- ADR-0033 — the commitment being extended.

## References

- ADR-0033 — the specification, inbound-only as written
- E-0088 — the parent epic
- `internal/verb/linkregion.go`, `linkrewrite.go`, `pathrewrite.go`, `archive.go`
