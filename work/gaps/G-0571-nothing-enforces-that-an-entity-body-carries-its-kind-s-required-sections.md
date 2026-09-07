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
one the committed body carries (ADR-0048). What stays open is everything a write
seam cannot see: the bodies already committed without a section, a body reaching
a commit without passing a verb, and `aiwf import`.

## Why it matters

The set is named "required" on five surfaces — the owned table, the `aiwf-add`
skill, the root help banner, the prose templates, and the design docs — and no
mechanism makes it true. An operator reading any of them is entitled to believe
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
edit. What would close them is a tree-side rule reading bodies it is not writing —
E-0084's push seam, or a rule with a baseline ledger — and that is where this gap
closes.

The gate at `internal/verb/add.go` sharpens the point. Handed a body whose required
section is present and empty, it refuses and tells the operator `aiwf check` will
block until the section is filled — which is true. An operator can satisfy that
refusal by deleting the heading rather than filling it, and then neither the gate
nor the check says anything. The stricter body is the one that is harder to land.

## Inherited obligation

M-0326 ships a standalone `aiwf check` rule reporting a `done` milestone whose
`## Release note` is absent or empty, rather than making it this machinery's
first consumer. Enforcing the declared set is the blast radius measured above
and belongs to this gap, so routing that one rule through it would have pulled
the cost forward into a milestone about something else. Whoever closes this gap
folds that rule into the general mechanism or records why it stays separate —
leaving it unexamined is the outcome the split was chosen to avoid.
