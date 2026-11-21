---
id: M-0365
title: Model operation identity in a ratified spine
status: draft
parent: E-0100
tdd: none
---
## Goal

A ratified spine for an operation's identity as ADR-0057 has it, from the record a verb or a
ritual writes through the copies a cherry-pick, a merge, a rebase or an amend makes, to what
`aiwf history` and `aiwf check` read; and a record of where today's `aiwf` departs from it.

## Closes

- (none)

## Context

ADR-0057's Context counts the operations this repository recorded twice and the routes that
recorded them. Today's `aiwf` writes no operation identity, so the spine and the binary differ
wherever the spine observes one.

## Acceptance criteria

## Out of scope

- Any change to `aiwf`.
- Commits that record several operations: ADR-0058's, modeled by the verb commit output
  milestone.

## Dependencies

- None among E-0100's milestones.

## References

- ADR-0057, and ADR-0056 for the identity's shape.
