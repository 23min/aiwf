---
id: G-0459
title: Event-shaped verbs append duplicate records on identical re-run
status: open
priority: medium
discovered_in: M-0281
---
## What's missing

Five event-shaped mutating verbs append a fresh record every time they run,
including when an identical record already exists in HEAD's history:

- `aiwf acknowledge mistag <id>`
- `aiwf authorize <id> --to <agent>`
- `aiwf promote <id> <state> --audit-only`
- `aiwf promote <id>/AC-<n> --phase <p> --audit-only`
- `aiwf cancel <id> --audit-only`

Each was measured by running the real binary twice with identical arguments: all
five exit 0 and land a second commit. `acknowledge illegal` carried the same
behavior until it was given a HEAD-walk duplicate guard; these five did not
receive the equivalent treatment, and each currently sits behind an allowlist
entry in `internal/policies/verb_result_noop_invariant.go` marked OPEN and
pointing here.

`acknowledge mistag` is the closest analogue: `check.WalkAcknowledgedMistags`
already walks HEAD for exactly these commits, so the detection capability exists
and is simply unused by the verb.

## Why it matters

Every repeat grows history with a record indistinguishable from the first, so
`aiwf history <id>` stops being a faithful account of distinct events. That is
the "re-running creates duplicates" smell, and the reason the duplicate guard on
`acknowledge illegal` was treated as a correctness fix rather than UX polish.

The decision is not mechanical, which is why it is filed rather than swept into
the convergence work:

- **The audit-only trio** exists to backfill a missing audit trail for state that
  was reached outside the verb. A second backfill for the same (entity, target
  state) records nothing new, so convergence looks right — but the verb's whole
  contract is "the entity is already at the target," so the guard must key on
  *an existing audit record*, not on entity state.
- **`authorize` is genuinely ambiguous.** A second grant may be a legitimate new
  event (a different agent, a different branch, a re-grant after a scope ended).
  Converging it to a no-op would be wrong in those cases. Worse, a silent no-op
  could mask the separate defect that a repeat currently leaves two
  simultaneously-`active` scopes on one entity with no check finding — see the
  companion gap. `authorize` may want a refusal or an FSM guard rather than
  convergence.
- **`acknowledge mistag`** is the one with an obvious answer: mirror the
  `acknowledge illegal` guard using the walker that already exists.

## Resolution shape

Decide per verb, not as a batch: mirror the existing HEAD-walk guard where a
repeat records nothing new, and refuse (or guard via the scope FSM) where a
repeat indicates operator confusion. Then replace each verb's OPEN allowlist
entry with either a converted implementation plus its test, or a by-design
reason. Resolve `authorize` only after the two-active-scopes question is settled.
