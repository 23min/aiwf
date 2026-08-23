---
id: D-0074
title: Refuse epic creation on a ritual branch; report unreachability at activation
status: accepted
---
> **Date:** 2026-08-23 · **Decided by:** human/peter

## Question

`aiwf add` never consults the current branch. `aiwf promote` does, for the two
activating transitions ADR-0010 pins to a parent branch. So an epic can be created
on a ritual branch and then cannot be activated: the promote refuses because the
expected branch is trunk, and moving to trunk puts the entity out of view, since
the file exists only on the branch where it was created.

Should the kernel catch this where it is created, where it is activated, or both?

## Decision

Both, with different jobs.

`aiwf add epic` refuses when the current branch carries a ritual rung —
`epic/`, `milestone/` or `patch/` per ADR-0010's grammar, classified by
`branchparse.RungOf`. The existing `--force --reason` on `add` is the bypass; no
new flag.

The predicate is the rung, not inequality with trunk, and the two are not
interchangeable. A repo whose trunk is `master` while the configured trunk name is
the default `main` has every branch unequal to trunk, so the inequality form
refuses every epic creation in that repo. Measured 2026-08-23: written that way,
the guard refused the scratch-epic seed in nine tests across two packages, each on
`master`. What strands an entity is being on a branch that merges later, which is
what the ritual rungs name.

The promote guard additionally reports when the entity is not reachable from the
expected branch, stating that fact and prescribing no remedy.

The guard is scoped to epics. Milestones are not covered: a milestone is created on
trunk and activated on its parent epic's branch, so what matters there is whether
trunk has been merged into that branch, which the wrap rituals' reconcile step
already does. Measured against this repo on 2026-08-23: the epic branch for the
active epic carries its milestone, and its history shows the reconcile merge that
put it there.

## Reasoning

`branchparse` is the canonical ADR-0010 branch grammar, written so consumers of
that grammar cannot drift apart. The guard routes through it rather than testing
prefixes itself; a third copy of the ritual-prefix set is the drift it exists to
prevent.

The cost of a refusal is paid where it fires. Refusing at creation costs nothing —
no entity exists yet, and the operator moves and retries. Refusing at activation
leaves an entity stranded on a branch, and every recovery from there is awkward:
merging the branch to trunk drags unrelated in-flight work with it, `--force`
records a sovereign override for what was an accident, and cherry-picking the
create commit leaves the entity on two branches with an unmeasured merge outcome.
Preventing is worth more than describing because the described state has no good
exit.

The promote half is kept anyway, for the population already in that state, and
because the guard's current message is wrong rather than merely unhelpful: it
sends the operator to a branch where the entity does not exist. It states the
unreachability and stops there, because which recovery is correct is not settled
and a message that recommends one would be asserting an answer this decision does
not have.

`--force` on `add` already means "bypass the born-complete empty-body gate", so it
now carries two meanings. That is accepted rather than overlooked: both are
sovereign overrides on the same verb, both already require a human actor and a
reason, and a second flag for one call site would cost more than the ambiguity. If
a third meaning ever arrives, split them then.

## Consequences

`aiwf add` gains branch awareness it has never had, and with it a dependency on
resolving the trunk name — the same resolution the promote guard already performs.
A workflow that deliberately drafted an epic while on a ritual branch now needs
`--force --reason` or a move to trunk.
