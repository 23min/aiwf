---
id: M-0167
title: Mutate-hunt corroboration for the structure-auditor policies
status: draft
parent: E-0042
tdd: none
---
## Deliverable

Every structure-auditor policy left in `grandfatherDark` after the first
milestone is accounted for: corroborated by a `mutate-hunt` run that confirms
the auditor catches a mutation of the structure it guards, and its kept ledger
entry carries a one-line note explaining why it stays (it fires only by
mutating a hardcoded Go structure, so a fixture cannot reach its firing path).

## Scope

The ~8 structure-auditors — for example `fsm-invariants`,
`trailer-order-matches-constants`, `closed-set-status-via-constants`,
`trailer-keys-via-constants`, `trailer-parser-uniqueness` — fire only when the
Go structure they audit is itself broken. Per policy, choose the cheapest
sound option (G-0259):

- Refactor to input-driven (accept the structure as a parameter so a broken
  fixture can be injected) where the refactor is cheap and clarifying. Any such
  refactor runs its own `wf-tdd-cycle` on this milestone branch.
- Otherwise corroborate via `mutate-hunt` and keep the `grandfatherDark` entry
  with the explanatory note.

## Outcome

The ledger is "burned down" in the honest sense: nothing dark is unaccounted
for. Every remaining entry is a structure-auditor with a documented reason and
mutate-hunt corroboration; every other entry has been deleted because a firing
fixture now lights it.

*Draft stub — acceptance criteria pinned when the milestone starts.*
