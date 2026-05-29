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

> **Why date and decided_by are in the body, not frontmatter.** aiwf core's frontmatter parser is strict — it only accepts the fields it validates (`id`, `title`, `status`, `relates_to`, `supersedes`, `superseded_by`). Putting `date:` or `decided_by:` in frontmatter would fail `aiwf check`. Keep them in the body header line above. The canonical commit timestamp and actor are also recoverable via `aiwf history D-NNN`. Delete this blockquote after copying.

## Question

What was being decided, and what made the answer non-obvious?

## Decision

What was decided. Imperative voice. One short paragraph.

## Reasoning

Why this answer over the alternatives. Name the alternatives, briefly say why they lost. Honest reasoning beats clever reasoning — future-you (or future-Claude) will thank you for the bullet that just says "we picked X because Y was unacceptably complicated to test."

## Consequences (optional)

If the decision implies follow-up work, migration cost, or downstream rules that future contributors should know — note them here. Skip if the decision is self-contained.
