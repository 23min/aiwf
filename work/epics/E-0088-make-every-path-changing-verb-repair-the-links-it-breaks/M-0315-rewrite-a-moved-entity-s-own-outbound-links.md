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
      status: met
      tdd_phase: done
    - id: AC-2
      title: A decision records whether ADR-0033 reaches outbound links
      status: met
      tdd_phase: done
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

## Work log

### AC-1 — The write path is shown safe or unsafe for editing the file being moved

**Safe.** A plan may move a file and write edited content at its new path;
failing after both land restores the worktree fully-old · commit 18f6e3e6d ·
internal/verb 636 passed, 1 skipped, 0 failed

The composition is not new — `aiwf move` and `aiwf retitle` each already emit an
`OpMove` of a file plus an `OpWrite` at that file's new path. What was untested
is the failure half for a single file: the existing rollback coverage pairs a
*directory* move with a rewrite of a file nested inside it, which exercises
different journal entries.

Correctness rests on the replay order rather than on the undo steps themselves.
`captureWrite` records the destination's state after the move has already put the
file there, so the journal reads "restore the destination's bytes", then "rename
the destination back". Replayed LIFO that leaves the original bytes at the
original path and nothing at the destination. Replayed in execution order it
leaves the *edited* bytes at the original path and a stray duplicate at the
destination — measured by inverting the loop in `applyTx.rollback`, which fires
all three of the test's assertions.

The answer clears AC-3 to rewrite a moved entity's own body rather than
reshaping it.

### AC-2 — A decision records whether ADR-0033 reaches outbound links

**Extension, not a separate commitment.** ADR-0046 records it, accepted, and
ADR-0033 cites it · commit 707db3c4d · internal/policies green

The argument the record rests on: ADR-0033's first bullet says "entity bodies
that point at it", but the boundary it actually polices is its second bullet —
"only files the loader owns" — and a moved entity's own body is such a file. The
rot class, the primitive and the discrimination rules are identical in both
directions; only the direction differs. The inbound-only wording tracks the
measurement that motivated the ADR, three of four `docs/adr` files linking into
`work/`, all inbound. Whether that scoping was deliberate is not recoverable
from the text, and the record says what it decides rather than claiming to
recover intent.

Extension over supersession, following the shape ADR-0041 already uses for
ADR-0030: both stay accepted, and the earlier record keeps its other four
bullets rather than being restated to avoid orphaning them. Supersession has
never been used in this repo, and a decision that is narrow rather than wrong is
a poor first use of it.

The evidence is a relationship check comparing two artefacts, so rewording
either document leaves it green while breaking the link turns it red. Measured
in both halves: dropping the citation from ADR-0033 fails on reachability,
returning ADR-0046 to proposed fails on settled-ness. `body-prose-id` already
covers a dangling citation, but never fires on a missing one, which is the half
that needed its own assertion.
