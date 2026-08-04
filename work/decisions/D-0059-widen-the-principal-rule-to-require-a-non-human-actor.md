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

Nor does it change hand-crafted history, which is the surface it might have been
expected to reach. The history-walking audit reports nothing at all for a commit
carrying no actor, so the half this widening adds is unreachable there too. The
payoff is coherence of record rather than new enforcement: the rule now says
what the design doc says, so the next reader of either is not left to work out
which one is authoritative.

That is a smaller gain than new enforcement would be, and it is still worth the
change. A rule whose condition is narrower than its documented statement is a
divergence that costs nothing until someone relies on the wrong one.

The alternative was to pin the current behavior and file the divergence for
later. That was rejected because it would write a verdict into the domain golden
that the design doc contradicts, and weaken the doc-sourced invariant to match
the code — resolving a coherence gap by softening the assertion, which the
parent epic's constraints forbid.

## Follow-ups

- The history-walking audit still expresses this rule in its narrower form,
  refusing a principal only alongside a human actor. The two are equivalent for
  every commit that audit can see, since it returns early for a commit with no
  actor — so this is a textual divergence, not a behavioral one, and it is
  recorded rather than filed.
