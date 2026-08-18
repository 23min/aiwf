---
id: D-0068
title: What counts as a defect the preflight pass missed
status: accepted
---
## Question

The preflight trial records what the pass missed. Without a definition, a later
session finding any defect anywhere must guess whether it counts, and the number
means whatever each session took it to mean.

## Decision

**A defect the pass missed is a defect in the specification, discovered between
the pass and the milestone's wrap, that a source in the pass's own corpus
contradicted — named by the `file:line` that would have shown it.**

The session that finds it records it, at the wrap gate D-0067 already
establishes. No new gate and no new ritual step.

No new term is minted. E-0086 bans vocabulary without a defect it fixes, and
plain description does the work here.

## Reasoning

Three clauses, each rejecting a wider reading.

**A source in the corpus contradicted it.** A design error nobody had recorded is
not a miss, because no amount of reading would have found it; an implementation
bug is not one either, because the pass never reads the implementation. The
narrow form makes the number measure the pass's coverage of its own corpus
rather than its omniscience, and it makes every instance falsifiable — the claim
is only admissible with the line that would have shown it. A definition nobody
can be wrong about produces a number nobody can trust.

**Between the pass and the wrap.** An unbounded window makes every future
discovery reopen a closed milestone's record, which is a mandate paid on every
subject forever. D-0053 accepts that shape only with a named retirement trigger,
and this one has no natural end. Bounding at wrap terminates and still catches
the defects that cost most, which are the ones found during implementation.

**The session that finds it.** Attribution to the pass rather than to the finder
would need someone to re-run the pass to know what it reported, which is the
cost the bound exists to avoid.

## Consequences

The number under-reports. A specification defect found after wrap never counts
against the pass that missed it, so the measured miss rate is a floor rather
than a true rate.

That is the deliberate trade: the number is comparable across consecutive
subjects, which is what the trial needs, rather than true, which it cannot
afford. A trial reading this number reads it as "at least this many", and a
rising floor is still a signal.

The falsifiable-instance requirement also means a session that cannot name the
line records nothing. Some real misses will go unrecorded because their
contradiction was diffuse rather than locatable. That loss is preferred to a
number inflated by claims no one can check.
