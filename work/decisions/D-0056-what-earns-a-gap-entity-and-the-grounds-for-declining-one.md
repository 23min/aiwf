---
id: D-0056
title: What earns a gap entity, and the grounds for declining one
status: proposed
---
## Question

Two shapes of proposed work keep arriving and keep being declined, and until now
the grounds were folded into individual close reasons rather than written once.
The first is the rule that nothing enforces — a defect report whose entire
content is "no detector exists for this." The second is the chore with no
judgment in it, where the work is real but every decision inside it is already
made.

Both are easy to file and hard to close, because each reads as an obvious
omission at the moment it is noticed. What was missing is a standing answer, so
that declining one is a policy rather than a mood.

## Decision

**A documented rule may stand without a chokepoint.** A gap whose whole content
is the absence of a detector is declined by default. Writing the rule down and
holding it at review is a legitimate terminal state, not an interim one.

**A chore with no judgment in it does not get a gap.** Where the work is
mechanical and every decision inside it is already settled, tracking it costs a
reader's attention that the fix itself would not.

**Declines rest on three grounds**, named at the close so the reasoning survives
the entity:

- **cost-per-subject** — the fix is a mandate that must be paid on every future
  subject, rather than a ban paid once
- **detector precision** — the check cannot separate the offending shape from
  its legitimate twin, so it would train its readers to ignore it
- **base rate** — the shape occurs too rarely for a permanent guard to earn the
  attention it takes to carry

## Reasoning

The alternative is to keep every such gap open, which was the status quo. It
loses on two counts. An open gap asserts intent, so a list of them overstates
what is actually planned and quietly devalues the entries that are real. And
because none of them ever becomes urgent, they accumulate: the reader who most
needs the list to mean something is the one who finds it full of items nobody
intends to take.

Declining case by case was the other alternative, and it is what has been
happening. It produces the right outcome and loses the reasoning — each close
re-derives grounds that are the same three every time, and a later reader sees a
verdict with no policy behind it.

The three grounds are not interchangeable, which is why they are named rather
than summarised as "not worth it." Cost-per-subject is an argument about the
fix, precision is an argument about the detector, and base rate is an argument
about the world. A decline citing the wrong one is a decline that will not
survive its own trigger.

Both arms share a prior: a rule that is written down and held at review is
already enforced, just not mechanically. Treating "unenforced by a machine" as
equivalent to "unenforced" is what makes a detector-shaped gap look obligatory,
and it is the step this decision refuses.

## Consequences

**Revisit on either trigger.** For a declined detector: a second live instance
where the absence actually cost something — one instance is an anecdote, and a
gap declined on base rate should reopen when the base rate is shown to be wrong.
For a declined chore: one large enough that someone needs to schedule it, at
which point the tracking earns its cost.

**Name the ground at close.** A `wontfix` citing none of the three is
indistinguishable from abandonment, and the trigger cannot be evaluated later
without knowing which argument it is meant to falsify.

**Precedent already applies this.** E-0076 was cancelled when the audit declined
the detector for two of its three members — G-0465 on cost-per-subject, G-0474
on base rate. G-0474 was subsequently closed on the same ground with its own
disposition recorded; G-0531 is the chore arm. This decision records the
practice rather than introducing it.

**This is not a licence to decline a defect.** The subject is the missing
*detector* and the mechanical *chore*. A gap describing behaviour that is wrong
is unaffected, whatever the cost of fixing it.
