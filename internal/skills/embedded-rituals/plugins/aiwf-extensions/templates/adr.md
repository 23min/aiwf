---
id: ADR-NNNN
title: <imperative, ≤ 60 chars>
status: proposed         # aiwf ADR statuses: proposed | accepted | superseded | rejected
supersedes: []           # optional: list of ADR ids this replaces
superseded_by:           # optional: ADR id that replaces this one
---

# ADR-NNNN — <imperative title>

> **Date:** YYYY-MM-DD · **Decided by:** <role or name>

> **Provenance.** This template follows [Michael Nygard's 2011 ADR pattern](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) — the de-facto standard adopted by `adr-tools`, the ThoughtWorks Technology Radar, and most ADR guides in the wild. Context → Decision → Consequences is the Nygard core; aiwf's status vocabulary is a tightened subset. Delete this blockquote after copying.
>
> **Why date and decided_by are in the body, not frontmatter.** aiwf core's frontmatter parser is strict — it only accepts the fields it validates (`id`, `title`, `status`, `supersedes`, `superseded_by`). Putting `date:` or `decided_by:` in frontmatter would fail `aiwf check`. Keep them in the body header line above. The canonical commit timestamp and actor are also recoverable via `aiwf history ADR-NNNN`.

## Status vocabulary (aiwf)

aiwf's ADR statuses are: `proposed | accepted | superseded | rejected`.

- `proposed` — written up, open for discussion or ratification.
- `accepted` — in force. Steady state.
- `superseded` — replaced by a later ADR. Set `superseded_by` on this one and `supersedes` on the new ADR. Never delete the file.
- `rejected` — proposed and explicitly turned down. Keep the file for the reasoning trail; do not re-use the number.

If you find yourself wanting `draft`, `pending`, or `partial` — those aren't aiwf ADR states. For incubating ideas, hold them in scratch until the proposal is real.

## Context

Why is this decision being made now? What forces — technical, organizational, regulatory, external — shape the choice? What alternatives are on the table? Keep this section honest: if an alternative was considered and rejected, name it and say why.

## Decision

State the decision in plain terms. One or two paragraphs. Imperative voice ("we use X for Y" rather than "it was decided that…"). If there are sub-decisions, bullet them. If the decision is phased, say so.

## Consequences

What follows from this decision? Positive and negative. Be specific about follow-up work, migration cost, things the team must do differently now. Cross-reference related ADRs, epics, or gaps where relevant.

## Validation (optional)

How will we know this decision still holds? A measurable signal, a periodic review cadence, or a trigger condition that should force a revisit. Leave the section out entirely if the decision doesn't need active validation.

## References (optional)

- Related ADRs: `ADR-NNNN`
- aiwf decisions: `D-NNN`
- Linked epics or milestones: `E-NN`, `M-NNN`
- External: docs, specs, RFCs
