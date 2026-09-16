---
id: G-0571
title: Nothing enforces that an entity body carries its kind's required sections
status: open
discovered_in: M-0305
---
## What's missing

No surface enforces that an entity body carries the sections its kind requires.
`entity-body-empty` reports a section that is present and empty; a heading absent
outright is skipped by design. The `aiwf add` gate consults the same helper, so it
inherits the same blind spot.

The scaffold does not cover the hole. For the born-complete kinds — adr, gap,
decision, contract — `aiwf add` refuses its own scaffold, because every scaffolded
heading is empty, so `--body` or `--body-file` is the only path to creating one and
that content replaces the scaffold wholesale.

M-0329 closed the write half. `aiwf add` refuses a body omitting a required
section for every kind, and both `aiwf edit-body` modes refuse a write that drops
one the committed body carries (ADR-0049). M-0331 closes the push: a push that
leaves a required section out of a body that carried it, or out of an entity
created without a verb, is refused there. What no seam covers is the bodies
already committed without a section, which ADR-0049 leaves unconverged, a branch
pushed once and merged on the server (G-0679), and `aiwf import`, excluded
pending G-0667.

## Why it matters

The set is named "required" on five surfaces — the owned table, the `aiwf-add`
skill, the root help banner, the prose templates, and the design docs — and no
mechanism made it true. An operator reading any of them is entitled to believe
a missing section would be caught.

The consequence is already in the tree, and a write-time refusal does not reach
it. Measured 2026-09-07 through the loader: 55 live entities omit at least one
required section, 109 omissions between them — 30 open gaps, 24 accepted
decisions, and one proposed epic — and no check reports any of them. They
concentrate in the born-complete kinds, which have no reachable scaffold, but the
hole is not theirs alone: the epic scaffold writes every required section, so
that one lost it after creation.

Closing it tree-wide would raise those 109 findings, 108 at error severity, which
is why E-0081 declined. Non-regression at the edit seams also means none of them
converges: an entity missing a section keeps it missing through every subsequent
edit, and the push seam asks the same question (ADR-0049), so no seam closes them.
Only a tree-side rule judging bodies against a baseline would, and none is built.

An emptiness refusal alone invites its own escape: an operator told to fill an
empty required section can delete the heading instead. Each seam now closes that
route — `aiwf add` refuses a body omitting the section, `aiwf edit-body` refuses a
write dropping one the committed body carried, and the push refuses a commit that
does the same.

## Inherited obligation

M-0326 ships a standalone `aiwf check` rule reporting a `done` milestone whose
`## Release note` is absent or empty, rather than making it this machinery's
first consumer. Enforcing the declared set is the blast radius measured above
and belongs to this gap, so routing that one rule through it would have pulled
the cost forward into a milestone about something else. The rule stays separate,
for reasons recorded in its own doc comment
(`internal/check/milestone_release_note.go`), so the split did not go unexamined.
