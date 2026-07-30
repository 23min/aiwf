---
id: E-0074
title: 'Event-shaped verbs converge: close the six OPEN NoOp allowlist entries'
status: proposed
---
## Goal

Finish the same-state convergence M-0281 started. That milestone converged twelve
operator-facing verbs and shipped
`internal/policies/verb_result_noop_invariant.go` to keep the convention from
rotting — but it left six allowlist entries marked OPEN, each an explicit IOU
rather than a by-design exemption. The policy therefore claims a discipline it
does not yet enforce everywhere, which is the same half-rolled-out condition
M-0281 was itself filed against.

Definition of done is mechanical and checkable: **zero OPEN entries in that
allowlist.** Every entry either becomes a real same-state NoOp assertion or is
rewritten with a by-design reason that survives review.

Addresses G-0458, G-0459 and G-0460.

## Scope

The six OPEN entries, in a forced order.

**First, G-0460** — a repeat `aiwf authorize <id> --to <agent>` exits 0, appends a
second commit, and leaves two simultaneously-active scopes on one entity with no
check finding. The `Authorize` allowlist entry says why this leads: convergence may
be the wrong fix, because a second grant can be a legitimate new event, and a
silent NoOp would mask the two-active-scopes defect. The scope invariant has to be
settled before any convergence decision on `authorize` means anything.

**Then G-0459** — four of the six entries. Five event-shaped verbs append a fresh
record on an identical re-run: `acknowledge mistag`, `authorize --to`,
`promote --audit-only`, `promote <id>/AC-N --phase --audit-only`, and
`cancel --audit-only`. `acknowledge illegal` already received a HEAD-walk duplicate
guard in M-0281; these did not. `acknowledge mistag` is the closest analogue —
`check.WalkAcknowledgedMistags` already walks HEAD for exactly the relevant
commits, so the detection capability exists and is unused. The `--audit-only` trio
needs a guard keyed on an existing audit *record*, because their precondition is
that the entity already sits at the target state.

**Last, G-0458** — `promote <id>/AC-N --phase <same-phase>` refuses via the
TDD-phase FSM. This is a decision, not a defect: the phase ladder is audit-bearing
evidence and the verb carries a `--tests` payload, so convergence needs a
deliberate metrics carve-out rather than a mechanical repeat. Resolve by converting
with that carve-out, or by rewriting the allowlist entry with a by-design reason.
It goes last because it cannot be decided by implementing it.

## Out of scope

- **Write-scope preconditions** — whether a verb should run at all against a
  frontmatter-dirty entity (G-0463, G-0466). Same layer, different axis, and it
  needs its own design decision; kept separate so a broad precondition change does
  not ride along with mechanical dedup work.
- **Prelude error-envelope uniformity** (G-0456). Shares the word "uniformity" and
  nothing else.
- **Retrofitting duplicate records already in history.** These guards prevent new
  duplicates; existing ones stay as the record of what happened.
