---
id: D-NNN
title: <imperative, ≤ 60 chars>
status: proposed         # aiwf decision statuses: proposed | accepted | superseded | rejected
relates_to: []           # optional: list of E-NN, M-NNN, ADR-NNNN this decision touches
supersedes: []           # optional: list of D-NNN this replaces
superseded_by:           # optional: D-NNN that replaces this
---

# D-NNN — <Decision Title>

> **Date:** YYYY-MM-DD · **Decided by:** <role or name>

aiwf decisions (`D-NNN`) capture project-scoped choices — typically tied to an epic or a milestone — that don't rise to the architectural weight of an ADR. Use this template for: scope cuts, sequencing decisions, mid-implementation pivots, deliberate trade-offs that the team should be able to find later.

If the decision is architectural, durable, and crosses multiple epics — author it as an ADR (`ADR-NNNN`) instead.

> **Put `date` and `decided_by` on the body header line above, not in frontmatter** — the strict frontmatter parser accepts only the fields it validates. Delete this blockquote after copying.

## Question

What was being decided, and what made the answer non-obvious?

## Decision

What was decided. Imperative voice. One short paragraph.

## Reasoning

Why this answer over the alternatives. Name the alternatives, briefly say why they lost. Honest reasoning beats clever reasoning — future-you (or future-Claude) will thank you for the bullet that just says "we picked X because Y was unacceptably complicated to test."

## Consequences (optional)

If the decision implies follow-up work, migration cost, or downstream rules that future contributors should know — note them here. Skip if the decision is self-contained.
