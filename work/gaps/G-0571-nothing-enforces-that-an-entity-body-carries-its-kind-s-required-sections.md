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
that content replaces the scaffold wholesale. `aiwf edit-body` consults nothing at
all. Measured: an epic created with the full scaffold, then given a body via
`aiwf edit-body --body-file` that omits `## Out of scope`, loses the section and
`aiwf check` reports zero errors.

## Why it matters

The set is named "required" on five surfaces — the owned table, the `aiwf-add`
skill, the root help banner, the prose templates, and the design docs — and no
mechanism makes it true. An operator reading any of them is entitled to believe
a missing section would be caught.

The consequence is already in the tree: 35 non-terminal gaps and 24 non-terminal
decisions are missing at least one section their kind requires, and the checks are
silent on every one. They concentrate in the born-complete kinds because those have
no reachable scaffold at all, but the hole is not theirs alone — an active epic in
this tree is missing `## Out of scope`, and epic is a kind whose scaffold does write
every required section. `aiwf edit-body` is how a body loses one afterwards.

Closing it tree-wide would raise 119 findings over 60 live entities — 118 of them
at error severity, plus one warning on an active epic missing `## Out of scope` —
which is why E-0081 declined to. The narrower option is a create-time refusal on
`aiwf add --body-file` and `aiwf edit-body --body-file`, which fires only on new
content and would raise none.

The gate at `internal/verb/add.go` sharpens the point. Handed a body whose required
section is present and empty, it refuses and tells the operator `aiwf check` will
block until the section is filled — which is true. An operator can satisfy that
refusal by deleting the heading rather than filling it, and then neither the gate
nor the check says anything. The stricter body is the one that is harder to land.
