---
id: D-0059
title: Widen the principal rule to require a non-human actor
status: proposed
relates_to:
    - M-0291
---
## Question

The provenance design doc states the principal rule as a required-together
pair: a principal is present exactly when a non-human actor is. The
implementation refuses a principal only when the actor is `human/`, so a
principal alongside no actor at all passes as coherent. The generated domain
test written for M-0291/AC-2 found the gap.

Closing it widens what the kernel refuses over history it did not previously
report, which is why it is a decision rather than a correction.

## Decision

Close it. The rule refuses a principal whenever the actor is not non-human,
which subsumes the human case and adds the absent one. The rule's name follows
the widened condition rather than continuing to name only the human half.

## Reasoning

The principal slot exists to name who a non-human actor acts for. A principal
with no actor at all names an accountability relationship with no agent in it —
not a state a reader could act on, so reporting it takes nothing away from an
operator.

The blast radius is measured rather than argued. No commit in this repo's
history carries a principal without an actor, and no verb path can produce one:
the CLI layer adds a principal only when the actor is non-human, and every verb
writes an actor. The widening therefore changes nothing reachable through
ordinary use.

What it does change is hand-crafted history. A consumer whose repo holds such a
commit sees a new error — but that commit is malformed by the design doc's own
definition, so surfacing it is the rule working rather than a regression.

The alternative was to pin the current behavior and file the divergence for
later. That was rejected because it would write a verdict into the domain golden
that the design doc contradicts, and weaken the doc-sourced invariant to match
the code — resolving a coherence gap by softening the assertion, which the
parent epic's constraints forbid.

## Follow-ups

- (none)
