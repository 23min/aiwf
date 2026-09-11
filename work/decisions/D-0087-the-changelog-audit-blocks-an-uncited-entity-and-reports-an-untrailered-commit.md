---
id: D-0087
title: The changelog audit blocks an uncited entity and reports an untrailered commit
status: proposed
relates_to:
    - E-0091
    - M-0330
    - G-0529
---
> **Date:** 2026-09-09 · **Decided by:** human/peter

## Question

The changelog audit reports two things: an entity whose shipped-surface change
nothing under `[Unreleased]` cites, and a shipped-surface commit carrying no
`aiwf-entity` trailer. Should both fail a release?

## Decision

The uncited entity is an error and fails the release. The untrailered commit is
reported at warning severity and does not.

## Reasoning

The two findings differ in what they establish, not in how much they matter.

An uncited entity is a proven violation of a rule the audit states: an entry
under `[Unreleased]` names the entity whose delta it describes. The audit reads
both sides and neither is ambiguous.

An untrailered commit establishes nothing about the changelog. The audit cannot
attribute it, so it cannot say whether an entry covers it. Failing a release
there reports the audit's own blind spot as the operator's fault, and the repair
it implies — rewriting a landed commit's trailers — is not available to them.
Reported, it names the commit for a human to place, and it grows the trailer
habit the blocking half depends on without holding a release hostage.

That habit is one release cycle old. Measured over the three ranges preceding
this decision, the untrailered share ran 20 of 20, then 38 of 52, then 2 of 11,
and is 0 of 18 on the current range. A blocking rule over a property that new
stops a release on the first lapse, and the audit is what gets switched off.

The uncited half has a baseline that clears: two entities owe an entry on the
current range and both close before any release — this epic's at its own wrap,
G-0659's as a line M-0330 writes. Nothing else in the range is uncited.

E-0091's risk row proposed landing the whole audit at warning severity and
escalating once the baseline was clean. That is overruled for the uncited half,
whose baseline is clean now, and adopted for the untrailered half, which nothing
the audit's operator controls can clean at the moment a release is cut.

Citation is a rule the audit imposes, not a property it observes, and it is
stated that way deliberately. The audit cannot prove a change is undocumented:
an entry describing it while naming no id is indistinguishable from no entry at
all. Thirteen percent of this file's entries name no id, and one of them sits in
the 0.33.0 section describing G-0635's change — which the audit reports as
uncited, correctly under the rule and wrongly under the stronger reading. Saying
"uncited" is what makes the finding provable; saying "undocumented" would claim
more than the audit can see.

## Consequences

An `[Unreleased]` entry now cites the entity it covers. That is a new obligation
on whoever writes one, and the wrap rituals already write the id into the
heading, so it costs nothing where a ritual runs and is the whole cost where one
does not.

The untrailered warning has no owner and no retirement trigger, which is
deliberate: it retires itself when the trailer habit reaches every shipped-surface
commit, and until then it is the only surface that says so.
