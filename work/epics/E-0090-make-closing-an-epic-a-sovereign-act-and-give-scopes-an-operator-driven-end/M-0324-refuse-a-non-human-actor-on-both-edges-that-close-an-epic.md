---
id: M-0324
title: Refuse a non-human actor on both edges that close an epic
status: draft
parent: E-0090
depends_on:
    - M-0323
tdd: required
acs:
    - id: AC-1
      title: Promoting an epic to done with a non-human actor is refused before any write
      status: open
    - id: AC-2
      title: Cancelling an epic with a non-human actor is refused at the verb, not the audit
      status: open
    - id: AC-3
      title: Every commit the widened audit fires on is ratified and check reports no error
      status: open
---
## Goal

Refuse a non-human actor on both edges that close an epic, at the verb, before anything is written — and ratify the one commit already in history that the widened audit will fire on.

## Closes

- G-0646 — both closing edges gated, with the `cancel` call site that makes the second entry enforceable.

## Context

`sovereignActShapes` holds one entry, epic `proposed → active`. Measured in a fixture, `aiwf promote <epic> done --actor ai/claude --principal human/fixture` exits 0, flips the status, writes no force trailer, and leaves a tree `aiwf check` calls clean. Commit `c030cb926` is that act, already in this repo's history.

Adding `active → done` to the closed set is a one-line change, because `promote` already calls the gate. `active → cancelled` is not: `requireHumanActorForSovereignAct` has a single call site, so a set entry alone would leave `cancel` silent while the transition-shaped history audit fired on the landed commit — refusal after the act, which is the record ADR-0040 exists to prevent.

## Acceptance criteria

### AC-1 — Promoting an epic to done with a non-human actor is refused before any write

`aiwf promote <epic> done` with a non-human actor returns an error naming the required `human/` actor, `HEAD` is unmoved, and the entity file is unchanged. A `human/` actor on the same transition still succeeds, so the gate is scoped to the actor rather than the edge.

### AC-2 — Cancelling an epic with a non-human actor is refused at the verb, not the audit

`aiwf cancel <epic>` with a non-human actor is refused before anything is written, by the same predicate. The distinguishing assertion is *where*: the refusal comes from the verb, with `HEAD` unmoved — not from a later `aiwf check` over a commit that already landed.

### AC-3 — Every commit the widened audit fires on is ratified and check reports no error

After the closed set widens, `aiwf check` over the tree reports no error-severity `fsm-history-consistent` finding. The historical acts the audit now reaches carry a ratification recorded by a human with a written reason.

Stated as a property of the tree rather than as a count, because widening the set is what determines which commits qualify, and a number written now would be a forecast.

## Constraints

- Sovereign-act shape is a property over legal transitions (D-0008); every entry added is FSM-legal and `TestSovereignActShapes_AllFSMLegal` stays green.
- A closed-set entry and the call site that enforces it land together (ADR-0040). Neither edge is added ahead of its verb-time refusal.
- `--force` stays human-only; the coherence guard at `verb.Apply` is not modified.
- The ratification is a sovereign act, so it is performed by a human and cannot be delegated.

## Design notes

The refusal message names only the human-run path. Offering `--force` there would be wrong every time it appeared: the message is reachable only for a non-human actor, and `verb.Apply` refuses that actor's force trailer anyway.

Two other consumers read the same closed set and widen automatically — the history audit in `internal/check/fsm_history_consistent.go`, and the static audit in `internal/policies/` that builds one regex per entry. Neither needs an edit.

## Out of scope

- Any change to the automatic scope-end at terminal promote.
- The identity substrate. The gate keys on a self-declared actor: an invocation that omits `--actor` inherits the human identity from `git config` and passes through. That property is shared with the shipped activation gate and is not addressed here.

## Dependencies

- M-0323 — the ADR decides whether the `cancel` edge is in scope at all, which AC-2 assumes.
