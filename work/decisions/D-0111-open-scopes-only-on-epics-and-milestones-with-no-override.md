---
id: D-0111
title: Open scopes only on epics and milestones, with no override
status: accepted
relates_to:
    - D-0007
---
> **Date:** 2026-11-24 · **Decided by:** human/peter

## Question

D-0007 limits `aiwf authorize` to epics and milestones, and its §Resolution adds that
`aiwf authorize <id> --to <agent> --force --reason "..."` remains available for the other kinds.
`authorizeOpen` in `internal/verb/authorize.go` refuses a gap, decision, contract or ADR with
`authorize-kind-not-allowed` before it reads `--force`, so that override does not exist (G-0780).
Does a human get an override for the kind rule?

## Decision

- `aiwf authorize <id> --to <agent>` opens a scope only when `<id>` is an epic or a milestone, and
  refuses a gap, decision, contract or ADR with `authorize-kind-not-allowed`.
- `--force --reason` does not lift that refusal.
- This decision supersedes D-0007, keeping its kind rule without the override.

## Reasoning

A scope lets an agent work on the entity it is opened on and the work under it: an epic's
milestones, a milestone's ACs, and the gaps discovered in either (`Tree.ReachesScope` in
`internal/tree/tree.go`). A gap, decision, contract or ADR has no work under it, so a scope on
one would let the agent act on that entity alone. For a decision or an ADR, that act is the
judgment the record exists to hold for a human, and the kernel does not otherwise keep it from an
agent: `internal/entity/sovereign.go` reserves only epic transitions for a human actor.

The alternative was to make the code match D-0007, with `--force --reason` lifting the refusal.
It lost because it builds a route nothing has needed, and the route opens the hole above. Every
commit on `main` whose `aiwf-verb` is `authorize` names an epic: on 2026-11-24, in the Linux
devcontainer, this command listed only epics in its third column.

```
git log main --format='%H%x09%(trailers:key=aiwf-verb,valueonly,separator=%x2C)%x09%(trailers:key=aiwf-entity,valueonly,separator=%x2C)' | awk -F'\t' '$2=="authorize"'
```

If a real need arises for one more kind, D-0007's likely case being
an agent maintaining a contract's schema, admitting that kind is a change to the allowed set,
made with a decision of its own.

## Consequences

- D-0007 is superseded. The code that names the kind rule's source names this decision.
- A test pins that `--force --reason` leaves the refusal in place, so a change that adds the
  override fails it.
