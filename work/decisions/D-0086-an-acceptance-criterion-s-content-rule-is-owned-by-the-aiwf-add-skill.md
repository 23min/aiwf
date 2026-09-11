---
id: D-0086
title: An acceptance criterion's content rule is owned by the aiwf-add skill
status: accepted
relates_to:
    - D-0085
    - G-0665
    - G-0659
---
> **Date:** 2026-09-08 · **Decided by:** human/peter

## Question

Which surface owns what an acceptance criterion carries — what its title names
and what its body holds?

## Decision

The `aiwf-add` skill's *What to write per kind* subsection. Every surface that
instructs a criterion body to be written or grown — the milestone-spec template,
`aiwfx-plan-milestones`, `aiwfx-start-milestone` and `aiwf-edit-body` — cites it
and states no rule of its own.

D-0085 does not reach this. That decision assigns milestone-spec *sections*, and
`## Acceptance criteria` is one it gives to the template as a section an author
fills while writing the spec — an assignment left untouched here. What sits under
that heading is a `### AC-N` sub-element with its own frontmatter record, so its
rule is a kind-body rule, and it is one row of the per-kind body table the
`aiwf-add` skill already carries for the other six kinds.

The evidence it takes to promote a criterion is a separate question with its own
owner. `aiwf-promote` states it, and nothing here moves it.

## Reasoning

Read literally, D-0085 returns the template: an AC body is first written while
the spec is being authored, and the template owns what an author fills. That
reading loses on D-0085's own premise about who is reading — the spec author has
the template open while scaffolding and the implementer does not re-open it. What
follows is that premise applied, not the survival argument D-0085 separately
disclaims: survival cannot discriminate *between* sections, since it is equally
poor for all of them, but it does answer who a template comment reaches. A
template comment is consumed at scaffold time, and the acceptance-criteria
comment block is gone from every finished spec — measured over the 329 milestone
specs in this tree, none retains it and none retains the AC placeholder either. So a rule placed there reaches the author
during the first fill and nobody afterwards, while G-0659 measures growth
continuing well past that point: of 225 criteria already substantial when
promoted `met`, 41 grew after the promote. A rule against growth placed where the
growth cannot see it binds nobody.

`aiwfx-plan-milestones` is where a body is first filled in practice, and it loses
on the writer set. D-0085's first-write tiebreak was built for sections with two
ritual writers. An AC body has an open-ended set: the planning ritual,
`aiwfx-start-milestone`'s recovery fallback, and `aiwf edit-body` during a review
round, which is inside no ritual at all. Each of the three cites the owner, so
the set can grow without the rule moving. First-write names the one moment at
which an anti-growth rule is least needed.

The verb skill is reachable from all of them, because each writer runs a verb and
the skill is what that verb dispatches. That is a property of the arrangement
rather than of the owner alone: `aiwf add ac` reaches the owner directly, and
`aiwf edit-body` — the verb the growth path runs — reaches it through a citation
this decision requires, in the section of that skill which explains AC-body
editing. Without that citation the argument would fail exactly where it matters
most, since the growth path would meet no rule.

G-0665 pre-rejects the verb skill — added there, the rule governs a command
rather than the artefact. That loses because `aiwf add ac` is how the artefact
comes into existence, and because the same subsection already governs artefacts
rather than commands for six other kinds. Any other owner forks that table so one
row lives elsewhere.

The assignment is a mandate, so it carries a mechanical owner rather than a
habit. Each citing surface is asserted to carry a citation naming both the owning
skill and its subsection, and the citation walk resolves that subsection against
the skill: deleting a pointer reports, retargeting one elsewhere in the same
skill reports, and renaming the owner's heading reports. Measured before those
checks landed, reverting the prose half of the change tripped nothing in the
repo.

The limits below are worth stating rather than discovering, because between them
they are most of what a review still has to carry.

The largest is that the owner can drop the rule while keeping the heading, and
every citation still resolves. That is a prose-content assertion over a shipped
surface, which D-0070 rules out; unlike the section retirement G-0530 closed
there is no name to ban, since prose can state or omit a body rule in unlimited
ways. A surface that keeps its citation and states a second rule beside it is
unreachable for the same reason.

The list of citing surfaces is a literal in the check rather than derived,
because "this surface states a rule about an AC body" has no machine shape. That
cuts both ways: a *fourth* surface that starts stating a rule is caught by
nothing, and an entry deleted from the list takes its surface out of the check
without anything reporting.

The check reads each citing file whole, so it holds that the citation exists in
the file, not that it sits where a reader of that surface will meet it. Scoping
it to a named section would be a heading-presence assertion, which is the class
D-0070 retires.

Each of them is carried at review, the same disposition D-0085 records for its
own equivalent.

What retires the mandate is the same thing that would retire D-0085's: a single
machine-readable source the rule can be derived from, so a surface's pointer is
generated rather than written and kept. The kernel's required-section table is
not that source today — for a milestone it carries `Goal` and
`Acceptance criteria`, and nothing about what sits under either.

The routing rule itself — that a new constraint on the body is written at the
owner — is not shipped. A materialized skill is regenerated by `aiwf update`, so
a consumer editing one loses the edit; the instruction addresses an aiwf
maintainer, and this record is where it lives.
