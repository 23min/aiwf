---
id: D-0093
title: A kind's body rules are owned by its template
status: accepted
relates_to:
    - D-0086
    - G-0680
    - G-0665
---
> **Date:** 2026-09-14 · **Decided by:** human/peter

## Question

Which surface owns what a kind's body sections should contain? D-0086 settled it
for an acceptance criterion and named the `aiwf-add` skill, but its question was
scoped to criteria. For the other five kinds nothing ruled, and both the skill's
*What to write per kind* subsection and each kind's template state rules. G-0680
measured the result: for most kinds the two disagree, and nothing keeps the rest
in step.

## Decision

A kind's body-content rules are owned by that kind's template. The `aiwf-add`
skill's per-kind entries route there and state no rule of their own. An acceptance
criterion keeps the owner D-0086 gave it, because it is the one kind with no
template file to hold the rule.

## Reasoning

The alternative was to extend D-0086's answer to all six kinds — the skill owns,
templates route. It loses on where the reader is standing. A template is the file
an author has open while writing the entity; the rule sits beside the section it
governs, and following it costs no lookup. Moving those rules into a verb skill
takes them out of the author's hands at the moment they are needed, and grows a
skill about *running a verb* by six kinds' worth of authoring prose.

It also loses on the evidence. Measured, the templates carry the fuller text and
the skill's one-line-per-kind entries are the ones that went stale — the gap
template asks for a pasted reproduction the skill's entry does not mention, and
the contract template names both ends of a contract where the skill names only the
consumer. Choosing the terser, drifted copy as owner would mean moving the better
text into it first, which is the same work with a worse resting place.

The acceptance-criterion exception is not a carve-out. Six kinds have templates
and a criterion does not — it is a sub-element of a milestone, scaffolded inside
the parent's body. The rule is "a kind's body rules live in its template", and a
criterion's rule lives in the skill because there is no template to put it in.
Read that way D-0086 is completed rather than contradicted.

## Consequences

The five per-kind paragraphs in the `aiwf-add` skill become routes. The templates
keep their text, except where the skill stated a rule the template did not: epic
`## Scope` and `## Out of scope` take theirs from the skill, so that a route is true
for every required section of every kind. A reader who learned the rules from the
skill must still reach them, so each route names where it points.

Nothing mechanical holds this. D-0070 retires prose assertions over shipped
surfaces, so no check can compare a template against a skill that no longer
restates it. What the routing buys is that the `aiwf-add` skill holds no second
copy; the planning and record rituals still hold theirs, tracked as G-0682. Keeping
it that way is a review obligation.

G-0680 is the work this decides.
