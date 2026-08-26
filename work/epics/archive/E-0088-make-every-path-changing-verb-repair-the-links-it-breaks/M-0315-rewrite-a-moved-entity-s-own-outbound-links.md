---
id: M-0315
title: Rewrite a moved entity's own outbound links
status: done
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
      status: met
      tdd_phase: done
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

### AC-3 — A moved entity's own relative links resolve after the move

`RewriteLinkDestinationsForMove` resolves a destination against the body's old
directory and renders it against the new one; wired into every mover that writes
its own body · commit a66a1a800 · internal/verb 640 passed, 1 skipped, 0 failed

The existing primitive could not reach this case by configuration. It rewrites
destinations naming a *moved target*, and outbound is the mirror: the target
stayed put and the linker moved, so the move index holds no entry and the
destination is left alone. The extension adds the second directory —
`RewriteLinkDestinations` now delegates with both paths equal, which is exactly
its previous behaviour.

The reach is a move that relocates a single file: a flat-file entity swept into
`archive/`, a milestone moved between epics. A move that relocates a whole
directory is excluded, and the exclusion is load-bearing rather than a
simplification — everything inside a moved directory comes along, including
files no verb enumerates, so those destinations still name the same content and
recomputing them breaks what worked. Measured: recomputing a dir-shaped
retitle rewrote an epic body's `[wrap](wrap.md)` to
`../E-NNNN-<old-slug>/wrap.md`, pointing at a directory that no longer exists.
Three fixtures pin it, one per verb that relocates a directory, each
driving a *nested* entity through the shared helper — the path the
dir-shape entity's own body never takes, since the verb rewrites that one
inline and excludes it.
The primitive cannot separate the two shapes from the paths it is given — a
directory rename and a milestone changing epics look identical — so the callers
that relocate a directory name it, and G-0623 carries the widening.

The `oldDir == newDir` guard bounds the blast radius, and the bound is the
reason the recompute is conditional rather than unconditional: re-rendering
returns a destination's *canonical* spelling, so a body carrying `./x.md` that
is not moving would be rewritten to `x.md` and pulled into an unrelated verb's
commit. Measured — dropping the guard makes `move` plan a write for a bystander
gap that neither moved nor links at anything that moved.

One predicate, `isRepoPathDestination`, decides whether a destination names a
file in the repo at all, and it runs on the bare path after any `?query` /
`#fragment` is split off. Five arms, each pinned by a fixture and each
preventing a distinct corruption: an anchor-only destination becoming
`(..#why-it-matters)`, a `mailto:` or `//host` or `/absolute` mangled into a
relative path, a rewrite inside `<angle brackets>` stripping the closer, and a
destination padded with spaces — legal in CommonMark, and defeating every
prefix test above it — emitted as two whitespace-separated tokens, which is not
an inline link at all, so the link would become literal text.
Running post-split is what closed G-0622, whose bug was a `://` test applied to
the whole destination — so a repo path carrying a URL in its query read as a
URL and was skipped.

Not repaired: a destination already broken before the move, and a dir-shaped
move's outbound links (G-0623). Both are recorded as ADR-0046 consequences, so
the record states the reach the code has rather than the reach the decision
argues for.

## Decisions made during implementation

- `ADR-0046` — path-link repair extends to a moved entity's own outbound links.
  Accepted, extending ADR-0033 rather than superseding it, following the shape
  ADR-0041 already uses for ADR-0030. AC-2 is this decision.
- The reach was narrowed to file-shaped moves after review measured that
  recomputing a dir-shaped move's destinations breaks links that worked. The
  narrowing is recorded in ADR-0046's Consequences, so the record states the
  reach the code has rather than the reach the argument supports.

## Validation

Measured at `e7e36e4ee`, after the third review round:

- `make check-fast` — exit 0. `go vet` across the untagged, `stress` and
  `testpins` tag sets; `go test -race -parallel 8 ./...` reporting `ok` for all
  71 packages with no `FAIL`; `golangci-lint run` clean.
- `AIWF_COVERAGE_BASE=epic/E-0088-make-every-path-changing-verb-repair-the-links-it-breaks make coverage-gate`
  — passes, including the firing-fixture meta-gate.
- `go test ./internal/verb/` — 644 passed, 1 skipped, 0 failed. The skip
  predates this milestone.
- `go build ./...` — exit 0.
- `aiwf check` — 0 errors, 8 warnings: six `terminal-entity-not-archived`, one
  `archive-sweep-pending`, one `provenance-untrailered-scope-undefined`. Two of
  the six are this milestone's own closures (G-0622) and additions awaiting the
  next sweep; the rest are inherited from mainline. The scope warning reports
  only that an unpushed branch has no upstream to audit against.

## Deferrals

- `G-0623` — outbound repair skips a move that relocates a whole directory. The
  primitive cannot separate that from a file changing directories using only the
  paths it is given, so the resolution needs the directory-level move as an
  input. The gap also records the wider half review measured: the suppression
  drops *inbound* relative repair for entities inside a moved directory, which
  is ADR-0033's ratified commitment unmet rather than only ADR-0046's reach
  falling short. Neither is a regression — both shapes behave exactly as they
  did before this milestone.
- The `colon == 0` boundary in `hasURIScheme` is statement-covered but
  mutation-surviving; only a destination beginning with `:` distinguishes it.
  Left to M-0316, whose subject is exactly this class.

## Reviewer notes

Three independent fresh-context review rounds, each returning request-changes,
each finding a defect the round before did not reach. All three are fixed and
pinned by a mutation run against the fix:

1. The not-a-path guard set was incomplete — `//host`, `/absolute` and
   `<angle-bracket>` destinations were corrupted, all three byte-identical
   before this milestone. Fixed by collapsing the question into one
   `isRepoPathDestination` predicate applied after the suffix split, which also
   closed G-0622: its bug was the same test applied to the whole destination, so
   a repo path carrying a URL in its query read as a URL.
2. A dir-shaped move rewrote destinations naming files that co-move with the
   directory. Fixed by narrowing the recompute to file-shaped moves.
3. That narrowing shipped unpinned — stubbing the whole suppression left the
   suite green, because the fixture pinned the verb's inline own-body rewrite
   rather than the shared helper, and carried no nested entity for the parameter
   to act on. Fixed by three fixtures, one per verb, each driving a nested
   entity through the helper. A second shape was still corrupted: CommonMark
   permits whitespace around a destination, which defeats every prefix test in
   the predicate.

The recurring failure is worth naming for whoever reviews the next milestone in
this epic: twice a guard shipped with a fixture that could not discriminate it,
and both times the test passed for a reason unrelated to what it claimed to pin.
Reading did not catch either; killing the specific mutant did.

Declined, so the next reviewer meets a decision rather than a blank:

- Four "URL stays untouched" assertions elsewhere in the suite sit where the
  linking file does not move, so they cannot fail. The property they claim is
  now pinned where it is distinguishable, and deleting the others would churn
  tests this milestone did not author.
- The `KindEpic || KindContract` predicate now has five copies across the movers.
  Extracting it is right and is not this milestone's work — it touches verbs
  outside the change-set, and the extraction wants its own review.

Not re-derived here: whether ADR-0033's inbound-only scoping was deliberate. The
text carries no evidence either way, and ADR-0046 records what it decides rather
than claiming to recover intent.
