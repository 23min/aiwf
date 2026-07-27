---
id: ADR-0036
title: Same-status FSM transitions converge to NoOp, not refusal
status: accepted
---
## Context

The kernel's per-kind FSM (`internal/entity/transition.go`) is deliberately
one-directional — no demote, no reversal. It lists only forward transitions, so
`ValidateTransition(X, X)` (a same-status "transition") is illegal by
construction, and `aiwf promote <id> <current-status>` returns an FSM error.
Four kernel surfaces encode "same-status is refused" as a *derived* consequence
of that FSM: the FSM itself, the legal-workflow spec's `terminalIllegal` cells,
the `internal/policies` negative-driver, and the `internal/stresstest` FSM
random-walk oracle (which also asserts "a successful promote lands exactly one
commit").

This surfaces as an operator UX smell (recorded in the scorecard's C2 finding):
re-running a promote — from a script, or having forgotten it already ran —
returns a confusing error on the second run instead of a clean no-op. The
kernel's atomic-single-commit and projection-validated guarantees already make
state convergence safe; the operator-facing outcome should match that safety.

## Decision

A mutating FSM verb (`promote`, `cancel`) whose requested target already equals
the entity's current status, with no other field changing, converges to a
**NoOp** — success, exit 0, a descriptive "already `<status>`" message, and
**zero commits** — instead of an FSM refusal.

- This does **not** reintroduce reversal. The FSM stays one-directional; you
  still cannot move backwards or demote. "Same-status" is the identity request —
  asking to stay where you are — now recognized as already-satisfied rather than
  an illegal transition. Self-loops remain absent from the FSM data; the NoOp is
  a verb-level short-circuit *above* `ValidateTransition`.
- The resolver-flag path is unaffected: a same-status promote carrying a
  resolver flag (gap `--by`, ADR `--superseded-by`) still backfills or refuses.
  The guard is gated on no resolver flag and no field change.
- For `cancel`, whose target is implicit (the kind's terminal end-state), the
  convergence condition is that the entity is already at *any* terminal status,
  not only cancel's own cancel-class terminal — a terminal entity is already
  disposed, so cancel has nothing to project (Option A). `cancel` of a
  success-terminal entity (a `done` epic, an `addressed` gap) is therefore a NoOp
  too; the message still names the actual state, so an operator cancelling a
  completed entity is informed, not misled. This is cancel's analog of promote's
  "target equals current."
- NoOp is a **first-class outcome** in the FSM-legality oracles. The correctness
  model becomes: *same-status FSM verb → NoOp (0 commits); a real mutation →
  exactly 1 commit.* The per-mutation-atomicity property is preserved — a NoOp
  is not a mutation.

## Consequences

- **Positive:** idempotent re-runs match declarative-tool convention (a promote
  to the state you are already in is a no-op, not an error); the most-cited
  operator smell (a promote run twice) is gone; behavior is uniform with the
  verbs that already NoOp (`archive`, `rewidth`, `contract bind`).
- **Cost:** the FSM-legality surfaces are updated to model NoOp deliberately,
  not silenced — the spec `terminalIllegal` doc boundary, the negative-driver
  probe target (non-self), the stresstest oracle (expect NoOp for same-status;
  allow 0 commits), and the mid-write-kill control (swapped to a genuine
  non-self illegal transition). These are correctness oracles changed on
  purpose.
- **Scope:** applies to the FSM-transition verbs (`promote`, `cancel`).
  Field-mutation verbs (`move`, `rename`, `retitle`, `acknowledge-illegal`)
  reach same-state NoOp with no FSM interaction, independent of this decision.
