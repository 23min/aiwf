---
id: M-0306
title: Check the prose templates contain the required set and fix the epic nesting
status: draft
parent: E-0081
depends_on:
    - M-0305
tdd: required
acs:
    - id: AC-1
      title: Each shipped prose template contains its kind's required sections at top level
      status: open
    - id: AC-2
      title: An epic drafted from the prose template carries out_of_scope in show JSON
      status: open
    - id: AC-3
      title: Templates and the add scaffold agree on whether a body carries a title H1
      status: open
    - id: AC-4
      title: The normative body-sections table states what the add scaffold writes
      status: open
---

## Goal

Make the shipped prose templates a checked superset of the owned section set, so a
body drafted by following the ritual satisfies the same rule as one the verb scaffolds.

## Context

The prose templates materialize into a consumer's `.claude/templates/` and are what
the planning rituals instruct an author to fill from. They are richer than the verb
scaffold by design — commentary, optional sections, fill-in guidance — and that part
is worth keeping. What is not worth keeping is their freedom to disagree with the
required set.

The epic template is the live instance. It places out-of-scope one heading level below
the flat `## Out of scope` that the check, the `aiwf add` scaffold, and the JSON body
map all name. Measured: every epic in this tree drafted from the prose template carries
no `out_of_scope` key in `aiwf show --format=json`; epics carrying the flat form do.
Because `entity-body-empty` skips an absent heading, neither side reports it.

A second disagreement sits on the same artifacts. The prose templates open with a
title H1 that the verb scaffold does not write and that no epic in this tree carries.
Two shipped surfaces, two answers, and nothing that decides.

Two axes of the template are already pinned against the real production oracle:
frontmatter decodes through `entity.Parse`, and prose is scanned by the real
`body-prose-id`. The section axis is the one with no such test.

## Approach

Extend the existing pattern rather than inventing one. The frontmatter test drives the
accepted-key set from the `Entity` struct through the real decoder; the section test
drives the required set from M-0305's owned definition, so it needs no second list to
maintain and a change to the set fails here without anyone remembering to update it.

Containment, not equality — the templates are a superset by design, and asserting
equality would forbid the optional sections that are the reason they exist. Heading
level is part of containment: a required section nested a level down does not count,
which is what makes the epic template fail on day one.

The H1 question is settled by evidence rather than preference. The tree's own answer
is visible — no epic carries one — and whichever way it goes, the deliverable is the
two surfaces agreeing and a test that keeps them agreeing.

## Acceptance criteria

### AC-1 — Each shipped prose template contains its kind's required sections at top level

For each shipped prose template, every section its kind's owned set names is present
as a top-level heading. The required set is read from M-0305's definition rather than
restated here, so adding a section to the set fails this test until the templates
follow. Optional and extra sections are permitted — the assertion is containment, not
equality. A required section present at a deeper heading level fails.

### AC-2 — An epic drafted from the prose template carries out_of_scope in show JSON

An epic body filled from the shipped epic template yields an `out_of_scope` key in
`aiwf show --format=json`. The test drives the real projection over a body built from
the shipped template bytes, not over a hand-written fixture that happens to be flat.

### AC-3 — Templates and the add scaffold agree on whether a body carries a title H1

The shipped prose templates and `entity.BodyTemplate` give the same answer on whether
an entity body opens with a title H1. A test pins the agreed answer and fails if
either surface changes alone. The criterion is the agreement, not which answer wins.

### AC-4 — The normative body-sections table states what the add scaffold writes

## Constraints

- Containment, not equality — the templates stay a superset and keep their commentary
  and optional sections.
- The required set is read from M-0305's owned definition; this milestone introduces
  no second copy of it.
- The templates keep materializing where D-0015 puts them; this milestone changes
  their content, not their destination.
- Every `SKILL.md` or template edit under the embedded rituals lands with its
  referencing structural test, per the repo's backstop policy.

## Design notes

The four milestone-template sections that duplicate structured data (G-0530) are not
touched here. Containment permits them, and whether they earn their place is a
separate judgment with its own evidence — one of them carries the wrap ritual's
commit-SHA trail.

Whether the containment check covers optional sections is deliberately unanswered:
only the required set is checked unless a second case appears.

## Surfaces touched

`internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/`,
`internal/policies/`.

## Out of scope

- The always-on guidance's scaffold instruction, which M-0307 routes.
- Whether the milestone template's duplicating sections should exist at all (G-0530).
- Collapsing the prose templates into the verb scaffold — four rituals read them.

## Dependencies

M-0305 — the owned section set this milestone checks the templates against.

## References

- G-0479 — epic template nests out-of-scope below the level three surfaces require
- D-0015 — ritual templates materialize to the templates dir
- E-0081 — parent epic

## Work log

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
