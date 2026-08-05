---
id: G-0550
title: Force with no actor at all is refused by neither rule set
status: open
discovered_in: M-0291
---
## What's missing

A commit carrying `aiwf-force` and no `aiwf-actor` at all is refused by nothing.
Both the verb-side rule and the history-walking audit test the actor for being
*non-human*, which an absent actor is not, so both read the set as coherent.

The design doc states the trigger differently: force is forbidden when the actor
"does not start with `human/`". An absent actor satisfies that, so the doc says
refuse and both implementations say accept.

## Why it matters

It is a sovereignty claim that nothing keeps, in a tree whose stated goal is
making sovereignty claims true. The usual backwards-compatibility argument does
not cover it either — both trailers predate the coherence rules, so a
force-without-actor commit is not a legacy shape that has to be tolerated.

Not reachable through any verb: the allow-gate refuses an empty actor before a
trailer set is assembled, and an omitted `--actor` resolves to the operator's
git identity. So this is about hand-crafted and imported history, which is
exactly the population the ratification path exists for.

The second half is the sharper one. The generated domain test cannot catch this,
because its "doc-sourced" invariants route the actor question through a helper
that encodes the *implementation's* notion of non-human rather than the doc's.
An invariant derived from the code it checks cannot falsify that code — the
back-fitting the test's own comment disclaims.

## Options

1. **Widen both rules to the doc's wording** and re-derive the invariant helper
   from the doc's phrasing rather than the implementation's. Closes the hole and
   the blind spot together.
2. **Widen the rules only.** Cheaper, leaves the domain test unable to detect a
   future regression of the same shape.
3. **Amend the design doc** to say the rules mean what they currently do. Honest
   if the absent-actor case is judged not worth refusing — but it is a
   claim-narrowing, and this epic's constraints forbid resolving a coherence gap
   by softening the assertion.

Option 1 is the lean. Option 3 is listed because it is a real answer, not
because it is a good one.

## Scope

Surfaced by the design lens of M-0291's wrap review. Neither the verb-side
narrowing that milestone performed nor the widening in D-0059 touched this case.
