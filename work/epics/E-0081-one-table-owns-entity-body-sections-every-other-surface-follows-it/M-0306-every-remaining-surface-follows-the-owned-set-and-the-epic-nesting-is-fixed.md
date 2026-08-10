---
id: M-0306
title: Every remaining surface follows the owned set, and the epic nesting is fixed
status: in_progress
parent: E-0081
depends_on:
    - M-0305
tdd: required
acs:
    - id: AC-1
      title: Each shipped prose template contains its kind's required sections at top level
      status: met
      tdd_phase: done
    - id: AC-2
      title: An epic drafted from the prose template carries out_of_scope in show JSON
      status: open
    - id: AC-3
      title: The templates mark the title heading optional, as the kernel treats it
      status: open
    - id: AC-4
      title: The normative body-sections table states what the add scaffold writes
      status: open
    - id: AC-5
      title: The self-check and coverage fixtures derive their bodies from the owned set
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

A second issue sits on the same artifacts, and it is not a disagreement. The prose
templates open with a title H1; the verb scaffold writes none. Both are correct: the
kernel treats the H1 as optional, and `aiwf retitle` implements that — it keeps a
canonical `# <id> — <title>` in sync when one is present and is a no-op when it is
not, leaving an operator-shaped heading alone rather than clobbering it. The tree
shows the same, unevenly: a minority of ADRs and decisions carry one, almost no gaps,
and no epic or milestone at all.

What is missing is the word "optional". A template that opens with the heading and
says nothing reads as mandatory to whoever fills it in, while the repo's own entities
mostly omit it.

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

The H1 needs no decision — the kernel already made it, and both surfaces already
comply. The deliverable is that a reader can tell: the templates say the heading is
optional, in the form they already use for their other optional sections, and a test
keeps the shipped surfaces from later contradicting the kernel's stance.

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

### AC-3 — The templates mark the title heading optional, as the kernel treats it

Each shipped prose template marks its opening `# <id> — <title>` heading as optional,
in the same form it already uses for its other optional sections. A test asserts the
marking is present on every template that carries the heading, so a later edit cannot
quietly present it as required.

The kernel's stance is not changed and no surface is made to match another. `aiwf
retitle` already treats the heading as optional — syncing a canonical one, no-oping
when absent, leaving a non-canonical one alone — and the scaffold writing none is that
stance, not a contradiction of it. What the templates lack is the word, and a reader
filling one in has no way to know the heading is theirs to skip.

### AC-4 — The normative body-sections table states what the add scaffold writes

`docs/design/design-decisions.md`'s body-sections table names, for each kind, exactly
the sections `aiwf add` scaffolds — which is what its own caption claims it lists. A
test compares the table against the owned set and fails if either moves alone.

Equality here, where AC-1 asserts containment, because the two surfaces answer
different questions. A prose template is a superset by design: it carries commentary
and optional sections a scaffold has no business writing. This table is captioned as
the scaffold's output, so a section it names that `aiwf add` does not write is not
extra detail — it is the caption being false. The milestone row is where that shows:
it lists five sections the rich template contributes, none of which the verb writes.

The doc's claim that bodies are not validated is corrected in the same change.
`entity-body-empty` reports an empty required section, and the born-complete kinds
refuse one at creation, so the sentence describes a kernel that has not existed since
those landed. That correction is prose and no test pins it; it is caught at review.

### AC-5 — The self-check and coverage fixtures derive their bodies from the owned set

`selfCheck*Body` in `internal/cli/doctor/selfcheck.go` and `bornCompleteFixtureBody`
in `internal/cellcoverage/fixture.go` stop spelling out section headings and render
them from `entity.RequiredSections` instead. Both build bodies for the four
born-complete kinds, which refuse a bare scaffold at creation and so need real prose
supplied.

This is the one criterion in the milestone that retires a surface rather than checking
one. Every other route here leaves a copy in place and adds a test to watch it; these
two can stop being copies, and then there is nothing to watch. That is the cheaper
outcome and the one the epic prefers wherever it is available.

The failure it forecloses is the worst-shaped in the inventory. Both fixtures are
correct today. If a kind's set later gains a section, each would build an entity
missing it — and because nothing reports an absent heading, `aiwf doctor --self-check`
would go on passing while creating exactly the defect the epic exists to prevent, in
the one place a consumer runs to ask whether their install is healthy.

## Constraints

- Containment, not equality, **for the prose templates** — they stay a superset and
  keep their commentary and optional sections. The normative table is the exception
  and AC-4 says why: it is captioned as the scaffold's output, so a section it names
  that the scaffold does not write makes the caption false rather than adding detail.
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
`docs/design/design-decisions.md`, `internal/cli/doctor/selfcheck.go`,
`internal/cellcoverage/fixture.go`, `internal/policies/`.

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
