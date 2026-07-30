---
id: G-0464
title: check predicates treat deferred ACs as non-terminal
status: open
priority: low
discovered_in: M-0281
---
## What's missing

Three check predicates decide whether an acceptance criterion is still in scope
for a body-completeness lint by testing whether its status is `cancelled`. The
AC FSM has two terminal statuses — `deferred` and `cancelled` — and neither
claims the criterion succeeded. A `deferred` AC is equally out of the milestone's
contract, but these predicates keep linting it:

- `internal/check/acs.go` (two sites)
- `internal/check/entity_body.go`

Measured: a `deferred` AC with a stub body fires `milestone-draft-incomplete-acs`
and keeps firing, because the only status that exempts it is `cancelled`.

## Why it matters

The operator's recourse is to cancel a criterion they deliberately deferred,
which loses the distinction between "we decided not to do this" and "we decided
to do it later" — the distinction the two terminal statuses exist to carry.

It also duplicates a fact the FSM already owns. `entity.IsTerminalACStatus`
derives terminality from `acTransitions`, so a predicate asking "is this AC
disposed?" can ask the FSM instead of naming one status and going stale when the
other changes. That is the same single-source discipline `IsTerminal`'s own doc
comment was written to protect.

## Scope

Pre-existing. It became reachable in ordinary use only when `aiwf cancel` on a
`deferred` AC stopped writing an FSM-illegal transition: that route used to
launder a deferred AC into `cancelled` and silence the lint as a side effect of a
bug.

## Proposed fix

Replace the three `== cancelled` comparisons with `entity.IsTerminalACStatus`.
The predicates then track the FSM rather than a hardcoded status, and both
removal-class terminals are treated alike.

Worth checking in the same pass whether other check rules ask the same question
the same way; the three above were found while investigating one lint, not by an
exhaustive sweep.
