---
id: M-0321
title: Declare kind-by-verb applicability and enforce cell totality
status: draft
parent: E-0089
depends_on:
    - M-0318
    - M-0319
tdd: required
acs:
    - id: AC-1
      title: The kind-by-verb applicability table is total over every pair, enforced
      status: open
    - id: AC-2
      title: An applicable coordinate with no cell fails a policy
      status: open
---
## Goal

Declare once, per kind and verb, whether the verb applies at all — and make a
missing cell at an applicable coordinate a policy failure rather than a silence.

## Context

Half the coordinate space carries no cell, and nothing distinguishes a coordinate
left silent on purpose from one nobody considered. Measured 2026-08-24: 99
coordinates, 49 declared. That is the condition this epic exists to end.

Not every hole deserves a cell. `cancel` has no meaning for a `tdd-phase` — the
sub-FSM has no cancelled state and `CancelTarget` returns nothing for it — so with
the target in the key, a cell cannot even be formed. D-0077 rules that whether a
verb applies to a kind is a fact about the verb's domain, not about state, so it is
declared once rather than repeated per state, and cell totality is then defined
over applicable rows only.

With `authorize` out of the cell table the verb set is `promote` and `cancel`, so
the applicability table is eight kinds by two verbs. Sixteen entries is small
enough to be total trivially, which is what stops it becoming the new place silence
hides.

## Acceptance criteria

### AC-1 — The kind-by-verb applicability table is total over every pair, enforced

Every kind-by-verb pair has an applicability entry, and a pair with none fails a
policy. A non-applicable entry carries the reason it does not apply. Adding a kind
or a legality verb without an entry reddens the suite.

### AC-2 — An applicable coordinate with no cell fails a policy

For every applicable kind-by-verb pair, every `FromState` of that kind carries at
least one cell. Deleting a cell fails the policy with a message naming the
coordinate. The count of coordinates the policy covers is recorded with the command
that produced it.

## Constraints

- **Both tables total, or neither counts.** An applicability table with a hole moves
  the silence rather than removing it, which is why AC-1 polices it in its own right.
- **A reason, not a flag.** A non-applicable entry says why; "false" alone
  reintroduces the ambiguity in a narrower field.
- **No cell written to satisfy the policy.** A coordinate that turns out to need a
  ruling is ruled, not defaulted.

## Design notes

The two tables answer different questions and it is worth keeping them visibly
distinct: applicability asks *does this verb mean anything for this kind*, the cell
table asks *from this state, toward this target, what happens*. Collapsing them
into one table with a sentinel outcome was considered and rejected in D-0077, on
the grounds that a key which does not fit a verb is better removed than padded.

## Out of scope

- Filling coordinates that need a ruling — those land with the milestone that owns
  their verb.
- Applicability for verbs outside the cell table's set.

## Dependencies

- M-0318 — totality is defined against the target-bearing key.
- M-0319 — the verb set is `promote` and `cancel` only once `authorize` has moved.

## References

- D-0077 — the ruling this implements
- `internal/entity/transition.go` — `CancelTarget`, the evidence for non-applicability
- `internal/workflows/spec/rules.go`
