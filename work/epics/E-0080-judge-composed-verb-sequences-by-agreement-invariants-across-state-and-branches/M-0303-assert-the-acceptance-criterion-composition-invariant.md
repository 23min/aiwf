---
id: M-0303
title: Assert the acceptance-criterion composition invariant
status: draft
parent: E-0080
depends_on:
    - M-0301
tdd: required
acs:
    - id: AC-1
      title: No AC is met under a tdd-required milestone whose phase is not done
      status: cancelled
    - id: AC-2
      title: The walker reaches a state where the invariant could fail
      status: open
---
## Goal

Assert the tree invariant G-0121 names by hand and nothing has ever checked
under composition: no acceptance criterion is `met` under a `tdd: required`
milestone whose `tdd_phase` is not `done`, after any legal verb sequence.

## Context

This is the one invariant G-0121 states explicitly, and it has stayed unasserted
for a structural reason rather than an oversight. E-0071 added `milestone tdd`
policy flips to the walker, which made the policy mutable mid-sequence by a real
verb — but the walker seeds no acceptance criteria, so a met-under-required
state was unreachable no matter how the policy moved.

M-0301 removes that. The state becomes reachable, and this milestone asserts the
property over it.

## Approach

Register the invariant into the oracle seam from M-0300, evaluated after every
step like the agreement properties. Unlike them, this one is not a comparison of
two observations — it reads a single tree and checks a relationship the kernel
already claims to maintain. That difference is fine: it still needs no model of
the correct verdict, which is the property D-0063 holds the oracle to.

The non-vacuity half matters more here than elsewhere. An invariant over a state
the walker cannot reach passes forever and reports nothing, which is exactly the
condition this milestone exists to end.

## Acceptance criteria

Each criterion below is asserted by the harness itself, not by review.

### AC-1 — No AC is met under a tdd-required milestone whose phase is not done

After every step of a composed sequence, the harness asserts that no acceptance
criterion is `met` while its parent milestone carries `tdd: required` and that
criterion's `tdd_phase` is anything other than `done`.

The invariant reads a relationship the kernel already claims to maintain — an
AC's status, its phase, and its milestone's policy — rather than a transition
table. Encoding the phase FSM here would put a second copy of
`entity.ValidateTransition` in the harness, drifting against the first.

A violation is a result, not a defect in the assertion. If a legal sequence
reaches this state, that is the composition defect G-0121 predicted, and it is
filed as its own gap rather than assertion-tuned away.

### AC-2 — The walker reaches a state where the invariant could fail

A test demonstrates that the walker reaches a state where this invariant could
fail — an acceptance criterion seeded, its phase moved, and its milestone's
`tdd` policy flipped underneath it — so a passing run is evidence rather than
silence.

This is the whole reason the invariant has stayed unasserted. E-0071 made the
policy mutable mid-sequence by a real verb, but the walker seeds no acceptance
criteria, so the state was unreachable and any assertion over it would have
passed forever while reporting nothing. D-0063 records that reasoning.

Reachability is asserted against the walker's real generated sequences, not
against a fixture built to order. A fixture would prove the invariant can fail;
only the walker proves this harness can find it.

## Constraints

- **No hand-written state.** The violating state, if reachable at all, is
  reached through real verbs. A hand-edited frontmatter that produces it would
  prove nothing about what the kernel permits.
- **The invariant does not encode the phase FSM.** It reads the relationship
  between an AC's status, its phase, and its milestone's policy — not a
  transition table that would drift against `entity.ValidateTransition`.
- **A violation found is a result, not a bug in the test.** If a legal sequence
  reaches a met-under-required state, that is the composition defect G-0121
  predicted; it gets its own gap rather than a weakened assertion.

## Design notes

- D-0063's Consequences record why this was unreachable: the walker seeds no
  acceptance criteria, so widening the mutation space is what unblocks it.
- E-0071 left this as its own milestone deliberately rather than folding it into
  the TDD-verb work.

## Surfaces touched

- `cmd/stresstest` — the oracle's registered invariants
- `internal/stresstest` — scenario drivers

## Out of scope

- Fixing any composition defect this finds. A violation is filed, not repaired
  here — the fix is kernel work with its own evidence bar.
- Other tree invariants. This milestone asserts the one G-0121 names; further
  invariants are cheap to add afterwards precisely because the oracle is
  invariant-shaped.

## Dependencies

- M-0301 — the walker must seed acceptance criteria before the state is
  reachable.
- M-0300 — the oracle seam this registers into.
- E-0071 — shipped the `milestone tdd` flip that makes the policy mutable
  mid-sequence.

## References

- G-0121 — states this invariant by hand under sub-gap 3.
- D-0063 — records why it was structurally unreachable.
- E-0071 — the `milestone tdd` mutation verb.

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
