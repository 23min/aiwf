---
id: M-0166
title: Firing fixtures for the easy-majority dark policies
status: in_progress
parent: E-0042
tdd: none
acs:
    - id: AC-1
      title: Firing fixtures for the single-site dark policies
      status: open
    - id: AC-2
      title: Firing fixtures for the multi-site dark policies (every dark site)
      status: open
    - id: AC-3
      title: Firing fixtures for acks-helper-lift; ledger reduced to fsm-invariants
      status: open
---
## Deliverable

A firing test for every dark construction **site** across the fixture-able
policies — a fixture (or input) that drives the policy to return ≥1 violation,
asserted end-to-end (not merely line-covered). Each policy whose every
construction site is covered is removed from the `grandfatherDark` ledger in
`internal/policies/firing_fixture_presence.go`.

## The meta-gate stays (decided empirically — see D-0025)

The firing-fixture meta-gate (`firing_fixture_presence` + `grandfatherDark`) is
**kept**, not retired. An adversarial empirical verify showed that the
alternative — a per-policy table-test ("fixture → ≥1 violation") — is strictly
*weaker*: darkness is **per-construction-site**, so a multi-site policy can hide
a dark site behind a single passing assertion (measured: ~106 sites across 51
policies, 19 multi-site), and the table-test loses the `-coverpkg` fail-closed
guard. So fixtures are written to **assert** (stronger than bare coverage) *and*
the coverage gate stays (per-site totality). They compose; neither alone is
complete.

## Scope

43 of the 44 ledger policies are fixture-able (verified: they scan files under
`root`). The 44th, `fsm-invariants`, is unreachable by any fixture (it
introspects compiled-in `entity` FSM symbols) and is handled in the sibling
milestone, not here.

The unit of work is the dark **site**, not the policy: a policy with multiple
construction sites (e.g. `trailer-order-matches-constants` has 6,
`acks-helper-lift` 16) needs fixtures covering every site, including the
defensive file-not-found fallbacks. Sizing follows the dark-site count, which is
larger than "43 fixtures."

## Mechanical evidence

`tdd: none` does not waive the evidence rule. Per policy: the firing test itself
(fails if the policy stops firing on its fixture), the meta-gate confirming
every construction site is now covered, and
`TestPolicy_FiringFixtureNoStaleAllowlist` (every removed id is genuinely lit).

## Acceptance criteria

Pinned after a per-site enumeration of the dark set.

### AC-1 — Firing fixtures for the single-site dark policies

### AC-2 — Firing fixtures for the multi-site dark policies (every dark site)

### AC-3 — Firing fixtures for acks-helper-lift; ledger reduced to fsm-invariants

