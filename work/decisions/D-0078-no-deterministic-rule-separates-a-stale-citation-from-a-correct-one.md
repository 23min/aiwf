---
id: D-0078
title: No deterministic rule separates a stale citation from a correct one
status: accepted
relates_to:
    - D-0075
    - G-0626
    - G-0628
---
> **Date:** 2026-08-25 · **Decided by:** human/peter

## Question

A closure invalidates claims in the records that cite the closed entity, and G-0628
asked for three layers to tell both sides. Two shipped: `aiwf promote` and
`aiwf cancel` print the live records that name what they just closed, and the wrap
rituals carry a step to judge them. The third was to be a standing `aiwf check`
rule — a body citing an entity that went terminal after that body was last
written — shipped at warning, swept, then promoted to error.

Does a deterministic rule over citations and dates separate a stale premise from a
correct citation well enough to ship?

## Decision

No, and none ships. G-0628's first two layers are the whole of the mechanism.

## Reasoning

Measured 2026-08-25 on main, counting distinct body-to-entity pairs throughout:

- **Any citation of a terminal entity** — 310 across 121 of 179 live gaps.
- **The rule as specified**, adding the body-write date — 45 across 38 live gaps;
  117 across 75 when every reporting kind is included. Reading all 45 the day
  before, roughly one in three named a claim its closure had falsified. The rest
  were dated measurements and past-tense narrative, both forms the standing
  guidance endorses.
- **The diff-scoped inversion** — a body written citing an entity already
  terminal — 2047 across 453 of 1757 body-write commits, 26% of every body write.

The third settles it. Its fires are reports: the three gaps filed this week name
six, two and two terminal entities, because a report about a closure necessarily
names what closed. Citing a terminal entity is how this tree references completed
work, so a rule keyed on the citation fires on the norm rather than the defect.
What separates them is whether a sentence presumes the entity is still live, which
is a reading of prose.

The specified rule also has no exit from its own noise. Shipping at warning, then
sweeping, then promoting to error assumes every fire is actionable; two in three
are not. No acknowledgement record was to be minted, and `aiwf acknowledge` carries
only `illegal` and `mistag`, so clearing a correct body's finding would mean
editing a record that is not wrong.

Alternatives considered and rejected:

- **Restricting to the planning kinds.** Three fires across one entity, all
  genuine, with a backlog small enough to repair by hand. Rejected because the
  flagship instance in G-0628 is a gap — one sequencing an implementation epic on
  an ADR that was rejected four days after the sentence was written — so the
  filter tracks the false-positive rate rather than the harm it claims to catch.
- **A query rather than a check.** No standing findings and no grandfathering, but
  an audit nobody runs is how this backlog accumulated.

## Consequences

- G-0628 closes on two layers. The closure is where the mechanism works, because
  it is the moment one person holds both sides.
- G-0626 has no deterministic detector, because its subject is the reading this
  decision rules out.
- D-0075 named diff-scoping as the shape worth building when a corpus-wide rule
  proves too noisy. Measured here, it is worse. Diff-scoping is what removes
  grandfathering, not what makes a signal precise; where the norm and the defect
  share a form, changing the scope moves neither apart.
- What would reopen this: a resolver reading structure rather than prose — a
  citation carried in a field, or in a body section whose meaning is fixed — so
  that a claim of dependence can be told from a mention.
