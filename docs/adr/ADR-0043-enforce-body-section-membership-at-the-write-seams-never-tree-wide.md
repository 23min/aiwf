---
id: ADR-0043
title: Enforce body-section membership at the write seams, never tree-wide
status: proposed
---
> **Date:** 2026-08-11 · **Decided by:** human/peter

## Context

`entity.RequiredSections` is the single definition of each kind's load-bearing body
sections, and every surface that states the set now derives from it or is tested
against it (E-0081). What none of them do is enforce it. `RequiredSections`' own doc
comment is explicit: *"'Required' names what the scaffold writes, not a guarantee
anything verifies."*

`entity-body-empty` reports a section that is present and empty; a heading absent
outright is skipped by design, and the `aiwf add` born-complete gate consults the same
helper, so it inherits the blind spot. The consequence is perverse rather than merely
incomplete: handed a body whose required section is present and empty, the gate refuses
and says `aiwf check` will block until it is filled — and an operator can satisfy that
refusal by deleting the heading instead of filling it, after which nothing says
anything. The stricter body is the one that is harder to land.

Measured on this tree: 217 of 564 gap bodies and 27 of 64 decision bodies omit at least
one section their kind requires. The rate declines over time but has not stopped — seven
of the last sixty-eight gaps, most recently G-0543. A body carrying invented headings
is accepted at creation for every kind and reported by nothing afterwards.

Enforcing membership across the tree would raise 119 findings over 60 live entities, 118
of them at error severity, which is why E-0081 declined to and recorded the hole as
G-0571 instead. That cost is what makes the placement question real: the property is
worth enforcing, and the obvious place to enforce it is the one that cannot be afforded.

## Decision

Body-section membership is enforced at the seams where body bytes are written, and
nowhere else.

**A violation is one thing:** a section the kind requires is not present as a top-level
`## ` heading. A heading nested below top level counts as absent, since that is already
how every reader sees it — `ParseBodySections` matches `## ` only, so a nested heading
produces no key in `aiwf show --format=json`. Sections beyond the required set are legal
and never flagged. Order is not enforced.

**Seam one — the verb.** A scan over the bytes a verb is about to write, called by every
body-supplying verb, refusing the write at error severity for every kind. This mirrors
`ScanBodyProseID`, which is the existing instance of the same shape and is excluded from
the projection path for the same reason: a projection models frontmatter, not body
content, so a body rule evaluated there reads stale or absent bytes.

**Seam two — the push.** A gate riding the commit range the provenance audit already
resolves, scoped to entities whose body content differs between the range base and HEAD.
Error severity. Scoping to *body changed* rather than *entity touched* is load-bearing: a
status promote or an archive sweep touches an entity file without touching its body, and
a touched-path scope would block both on debt they did not create.

**Neither rule joins `check.Run`.** `aiwf check`'s tree-wide output is unchanged.

Two seams rather than one because they answer different questions. The verb seam refuses
at the moment of the mistake, with the entity in hand and no commit yet written. The push
seam is the authority, and it is not redundant: a body can reach a commit without passing
any verb. One such path ships today — the wrap-milestone ritual commits the milestone
spec with `git commit` and hand-written trailers, so the verb seam never sees those bytes
and the provenance rule, which keys on trailer presence, passes them. This is the
kernel's standing layering — fire as early as the class allows, and let the push be
authoritative — applied to one property.

**Forward-only is a consequence here, not a policy.** Neither seam reads an entity it is
not writing, or whose body the push did not change. There is no age test, no grandfather
ledger, and no suppression rule that could be misapplied to a new entity. The existing
violations are out of scope because nothing looks at them, not because something decided
to forgive them.

## Consequences

- The 119 existing violations are never reported by anything. They remain debt; G-0571
  holds the measurement, and closing them is a migration, not a check.
- A status promote, a retitle, an archive sweep, or a reallocate on a non-conforming
  entity is unaffected — body unchanged means out of scope at both seams.
- No workflow is blocked that was previously available. An author who does not yet know a
  section's content keeps the heading and leaves it empty, which is the existing
  `entity-body-empty` finding — a warning for epic and milestone, an error for the
  born-complete kinds. Membership never requires content, only the heading.
- The wrap-milestone ritual's plain-`git commit` write of the milestone spec is covered by
  the push seam rather than the verb seam. Routing that write through `aiwf edit-body`
  would make every body write verb-mediated and is worth doing on its own merits — the
  current commit also bundles an entity body edit with `ROADMAP.md`, against the
  one-verb-one-commit invariant — but it is not what makes this decision correct.
- When the provenance audit's range is undefined — no upstream configured and no
  `--since` — the audit skips, and the push seam skips with it. This is inherited, not
  introduced. `aiwf check` on CI, where the ref resolves, is the backstop.
- A future body-writing verb that does not call the scan is unenforced at seam one, and
  nothing forces the call. The push seam still covers it, so the failure degrades to late
  feedback rather than no feedback.
- The scan reads `RequiredSections` and nothing else. The rich prose templates are not an
  input, so a template edit cannot change what is enforced.
- `--force` bypasses both seams, as it bypasses every other refusal.

## Validation

The measurements this decision rests on, all taken 2026-08-11 against this tree with a
binary built from it:

- Membership unenforced: a body with invented top-level headings is accepted by
  `aiwf add` for all six kinds and draws no finding from `aiwf check`.
- The tree's non-conformance: 217 of 564 gaps, 27 of 64 decisions, 0 of 41 ADRs.
- Template availability does not predict conformance — ADR carries the richest prose
  template and has no violations; decision carries an equally-resolving one and has more
  recent violations than gap, which has no template at all. This is why the enforcement
  seam, not the authoring instruction, is the load-bearing surface.
- No entity in the tree carries unfilled template placeholder text, which is why
  placeholder detection is deliberately not part of the rule.
- `UntrailedCommit` already carries the touched paths for every commit in the audited
  range, so the push seam needs no base-ref machinery of its own.

## References

- G-0571 — the hole this closes, and the source of the 119-finding measurement
- ADR-0029 — verb shape correctness comes from pre-write projection; this decision states
  why a body rule cannot ride that path and takes the byte-scan route instead
- ADR-0034 — the per-kind applicability pattern this follows: one predicate as the single
  source, one rule, one firing fixture
- D-0054 — keep the reasoning, derive the facts; the reason this rule replaces
  authoring instructions rather than joining them
- E-0081 — the epic that gave the section set one owner, and deliberately excluded
  enforcement
