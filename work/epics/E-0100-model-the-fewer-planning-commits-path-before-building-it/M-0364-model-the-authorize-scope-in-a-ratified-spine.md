---
id: M-0364
title: Model the authorize scope in a ratified spine
status: draft
parent: E-0100
tdd: none
---
## Goal

A ratified spine for one agent's delegation scope as ADR-0056 has it, and a record of where
today's `aiwf` departs from it.

## Closes

- (none)

## Context

The pilot's scope spine in `docs/research/fewer-planning-commits/verifier-kit-pilot/scope/` has
the states none, active and ended and the events `authorize`, `act`, `end`, `rebase` and `check`.
It was written knowing G-0751 and G-0752, without an audit or readings, and has no pause, resume
or copied opening commit (G-0755). Its adapter plays those events with `aiwf` in disposable
repositories.

## Acceptance criteria

## Constraints

- The pilot's scope spine is not a reading: the readings start from the chapter alone.

## Out of scope

- Any change to `aiwf`.

## Dependencies

- None among E-0100's milestones.
- E-0100's open question on installing verifier-kit into the repository, decided when this
  milestone starts.

## References

- ADR-0056.
- G-0751, G-0752 and G-0755.
