---
id: D-0042
title: rewidth reference sweep stays active-tree-only by design
status: accepted
---
> **Date:** 2026-07-19 · **Decided by:** human/peter

## Question

`reallocate` rewrites cross-references everywhere including `archive/`
subtrees; `rewidth` deliberately excludes archives at every level of its walk.
Both mutate ids — should `rewidth` match `reallocate`'s archive-inclusive
sweep, or is the divergence sound? Non-obvious because "two id-mutating verbs,
two different sweep guarantees" reads like drift until the risk asymmetry is
named.

## Decision

Keep `rewidth`'s reference sweep active-tree-only. The divergence is
principled, not accidental, and is now documented rather than implicit.

## Reasoning

- The asymmetry: `reallocate` changes an id's *identity* — an unrewritten
  archived reference afterward points at the wrong entity, silently. `rewidth`
  changes only *formatting* — parsers tolerate narrow legacy widths on input
  (ADR-0008), so an unrewritten archived reference still resolves to the
  correct entity. One sweep guards meaning; the other would guard polish.
- Widening was rejected because it rewrites terminal, settled archive files
  for cosmetic gain, enlarges what `rewidth --apply` touches (the audit warned
  this is a bigger behavior change than it looks), and buys a guarantee the
  pinned width-tolerance already provides.
- Cost accepted: tooling that greps for canonical-width ids will still meet
  narrow ids inside `archive/`.

## Consequences

- Revisit trigger: any future decision to drop input width-tolerance
  (ADR-0008's parser leniency) must resurface this — without that tolerance,
  archived narrow-width references would stop resolving.
