---
id: D-0061
title: Retire the sovereign-dispatcher policy; the force rule lives at the apply seam
status: proposed
relates_to:
    - M-0293
    - ADR-0040
---
## Context

`PolicySovereignDispatchersGuardHumanActor` asserted that every CLI dispatcher
parsing a sovereign act references the human-actor guard. Its predicate was a
substring search over function bodies, which every dispatcher satisfied through
a flag-help string naming the actor shape — measured in G-0534, where all four
passed on help text and no guard in code.

G-0534 recorded that the fix turned on an unsettled question: what is the
dispatcher layer supposed to assert that the verb layer does not? It named
three options — narrow the predicate and write the guards, retire the policy, or
keep it and document the weaker claim — and declined to lean until that question
was answered.

M-0293 answered it by building option one and measuring the result.

## Decision

The policy is retired, along with the dispatcher-layer guard written to satisfy
it. The sovereign-force rule is enforced at `verb.Apply` and nowhere else.

The dispatcher layer cannot assert this rule, and the reason is structural
rather than a matter of effort. Whether `--force` denotes a sovereign act
depends on verb and on tree state, and the dispatcher has neither:

- A converging request emits no plan and therefore no force trailer. ADR-0036
  makes exit 0 with no commit the contract for an already-satisfied request, and
  ADR-0040 states the case directly: *"a converging verb writes no commit, so it
  emits no trailer, so it has no coherence to violate."* A flag-keyed pre-check
  refuses it anyway.
- `aiwf add --force` bypasses the born-complete body gate and is inert on kinds
  that have none, so the flag records no sovereign act there and is accepted
  from any actor.
- A check placed before the tree is loaded also speaks ahead of the rules that
  need it, answering with the force refusal where the principal rule or a
  not-found error is the operator's real problem.

Moving the check later removes each of those, and also removes its only stated
benefit — refusing before the repo lock and the tree load. There is no position
at that layer where the check is both correct and worth its cost.

## Consequences

`verb.Apply` remains the single enforcement point, which is what ADR-0040 chose
it for: any caller reaching it is covered without being enumerated. Nothing that
was enforced stops being enforced — a forced act by a non-human actor is refused
before anything is written, with `HEAD` unmoved, demonstrated against the shipped
binary.

The dispatcher layer keeps no assertion about force. A future contributor asking
whether one belongs there meets this decision rather than a green chokepoint
that cannot fail.

R-AUDIT-0070 now cites `verb.Apply` and `CheckForceTrailerCoherence` rather than
the retired policy. G-0534 closes on this decision; its option two required a
recorded rationale rather than a quiet removal, and the caution G-0535 records —
that a policy retired because nothing else passed keeps its name and loses its
subject — is answered by the measurement above rather than by argument.

The trailer-not-flag distinction the reasoning turns on is pinned by
`TestAddForceIsInertWithoutAGateToBypass` (`internal/verb`), and stated in
`docs/design/design-decisions.md`.
