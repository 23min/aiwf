---
id: M-0166
title: Firing fixtures for the easy-majority dark policies
status: draft
parent: E-0042
tdd: none
---
## Deliverable

A firing test beside each "easy-majority" dark policy — a policy that fires by
scanning a fixture, file, or tree — so its `Violation` construction line is
covered by a test that drives it to return at least one violation. Each policy
so covered is removed from the `grandfatherDark` ledger in
`internal/policies/firing_fixture_presence.go`.

## Scope

The dark policies split into two families (G-0259):

- **Easy majority** — fixture/file/tree-scanning policies whose firing path is
  reached by handing them a crafted input. These are this milestone's target:
  write a fixture that violates the rule, assert the policy returns ≥1 violation.
- **Structure-auditors** — policies that fire only by mutating a hardcoded Go
  structure. Those are out of scope here; they are the next milestone's work.

The exact split is established by classifying the 44 ledger entries before the
acceptance criteria are pinned.

## Mechanical evidence

`tdd: none` does not waive the evidence rule. Per policy, the evidence is the
firing test itself (passes only because the policy fires on the fixture), plus
two existing gates that confirm the ledger shrank honestly: the firing-fixture
meta-gate (the construction line is now covered) and
`TestPolicy_FiringFixtureNoStaleAllowlist` (every removed id is genuinely lit).

## Acceptance criteria

Pinned after the classification pass.
