---
id: M-0305
title: Single-source the per-kind section set and retire Approach
status: in_progress
parent: E-0081
tdd: required
acs:
    - id: AC-1
      title: The per-kind section set has one definition the add scaffold derives from
      status: met
      tdd_phase: done
    - id: AC-2
      title: The JSON body-map help text is checked against that definition
      status: met
      tdd_phase: done
    - id: AC-3
      title: An absent required section produces a finding naming it
      status: cancelled
    - id: AC-4
      title: Approach is retired from the table and the doc surfaces follow
      status: met
      tdd_phase: done
---

## Goal

Give the per-kind body-section set one definition that every other surface derives
from or is checked against.

## Context

`requiredSectionsByKind` in `internal/check/entity_body.go` names each kind's
load-bearing sections. `entity.BodyTemplate` in `internal/entity/serialize.go` writes
the scaffold `aiwf add` commits. The two are independent literals, and for milestone
they disagree: the check names `Goal`, `Approach`, `Acceptance criteria`; the scaffold
writes `Goal`, `Acceptance criteria`. The root command's help text describing the
`show --format=json` body map restates the set a third time.

The disagreement is invisible because the rule cannot see it. `EmptyRequiredSections`
walks the sections a body actually has and reports the ones that are empty; a section
whose heading is absent is skipped, a stance its own doc comment takes deliberately.
So the check polices emptiness, never membership — and membership is what the two
literals disagreed about, which is why the drift ran for as long as it did.

Nothing else enforces membership either — not the scaffold, which the born-complete
kinds cannot use, and not `aiwf edit-body`, which consults nothing. That is a real
hole and a separate one; it is tracked as G-0571 rather than closed here.

## Approach

Give the set one owner and have the other surfaces read it rather than restate it.
The scaffold is the surface that must derive — a literal beside the table is what let
the two drift in the first place. The help text is checked rather than generated,
since it is prose describing a projection rather than a copy of the set.

The rule itself is left alone. Firing on an absent heading would indict live entities
across the tree at a cost this milestone is not the place to impose; G-0571 carries
the question and the measurement.

The milestone set loses `Approach`. It reached the table from the body of the
milestone that introduced the rule, never appeared in the prose template the planning
ritual fills, and is absent from the normative statement of what a milestone body
carries. Retiring it costs nothing measurable: the rule reads the names the set holds
and never enumerates the rest, so the bodies already carrying the section keep it and
produce no finding.

## Acceptance criteria

### AC-1 — The per-kind section set has one definition the add scaffold derives from

For every kind, the sections `entity.BodyTemplate` writes equal the sections the
owned set names, in order. The test drives both from the owned definition, so a kind
added to one and not the other fails rather than passing by matching copies.

### AC-2 — The JSON body-map help text is checked against that definition

The root command's help text describing the `show --format=json` body map names, for
each kind, the slugified form of that kind's owned section set. A section added to or
removed from the set without the help text following fails the test.

### AC-3 — An absent required section produces a finding naming it

An entity body missing a required section produces a finding that names the absent
section and the entity. The path runs through the helper the `aiwf add` verb gate and
the `entity-body-empty` rule share, so both answer the same way for the same body.
A body carrying every required section produces no such finding.

### AC-4 — Approach is retired from the table and the doc surfaces follow

The milestone entry in the owned set does not name `Approach`. No allowlist,
exemption ledger, or grandfather entry is introduced by this milestone. Every
non-archived, non-terminal milestone carries every section the set names, both
before the removal and after, so the change is finding-neutral. The bodies that
already carry `## Approach` keep it and produce no finding: the rule looks up the
names the set holds and never enumerates the rest, so a retired section is inert
rather than unexpected.

`RequiredSections` and `EmptyRequiredSections` say in their doc comments which half
of the guarantee each carries — the scaffold writes the set, the check polices only
emptiness of what a body has. Nothing verifies membership, and the comments name
that rather than leaving a reader to infer it from the word "required".

Every remaining surface that states the set follows in the same change:

- `internal/skills/embedded/aiwf-add` — a required-body-sections table and a prose
  paragraph per kind. Both are shipped to consumer repos and are what an authoring
  assistant reads, so leaving either would reproduce the drift this milestone exists
  to end. The table is pinned against the owned set; the prose is not, and a section
  named there is caught at review rather than mechanically.
- `internal/skills/embedded/aiwf-show` — the body-key table. Its milestone row drops
  `approach`. Its gap row names `what_s_missing`, the slug `SectionSlug` derives from
  `What's missing`, rather than `whats_missing`, which no `aiwf show` envelope
  carries. A test asserts the row names every slug the owned set implies; it does not
  constrain the keys beyond them, which that table legitimately carries.
- `internal/check/entity_body.go` — the package doc comment restated all six kinds in
  prose beside the literal this milestone removed, and now names neither.

The scaffold's rendered bytes are asserted per kind, not just its heading names.
Comparing parsed headings against the table they are rendered from can only fail on a
round trip; comparing the bytes catches a display-form edit and the loss of the blank
line between headings, neither of which any suite currently sees.

## Constraints

- No new chokepoint, and no change to what `aiwf check` reports. This milestone moves
  and prunes a definition; the rule reading it keeps its existing behaviour.
- Layering direction is already policed; wherever the owned set lands, the existing
  layering rule decides the direction, not convenience.
- No grandfather allowlist survives this milestone.

## Design notes

Where the set lives is open. `internal/check` reads it today, but `entity.BodyTemplate`
must derive from it and `entity` sits below `check`, so the table likely moves down
rather than the dependency moving up. Settled against the layering rule during AC-1,
not guessed here.

## Surfaces touched

`internal/entity/required_sections.go`, `internal/entity/serialize.go`,
`internal/check/entity_body.go`, `internal/verb/ac.go`, `internal/cli/root.go`,
`internal/cli/integration/`, `internal/policies/`,
`internal/skills/embedded/aiwf-add/SKILL.md`,
`internal/skills/embedded/aiwf-show/SKILL.md`, `docs/design/growth.md`.

## Out of scope

- Enforcing membership on any path — the tree-wide check, and a create-time refusal
  alike. Both are G-0571's subject, and it holds the measurement.
- The shipped prose templates, which M-0306 reconciles against the owned set.
- The always-on guidance's scaffold instruction, which M-0307 routes.
- Whether the milestone template's duplicating sections should exist at all (G-0530).

## Dependencies

None. First milestone of E-0081.

## References

- G-0482 — the milestone template and the required set disagreed about `Approach`
- G-0571 — nothing enforces that a body carries its kind's required sections
- E-0081 — parent epic, carrying the surface inventory

## Work log

### AC-1 — The per-kind section set has one definition the add scaffold derives from

Table moved to `internal/entity`; `BodyTemplate` renders from it, the check rule
reads it, and a third copy in `BodyTemplate`'s own test was deleted rather than
updated · commit 5227fef7f · tests all green

Single-sourcing is guaranteed by construction once the scaffold renders from the
table, so the AC's test can only fail on a render/parse round trip. The table's
display form is a separate axis and this left it unpinned: a case-only edit to a
section name, or the loss of the blank line between headings, passes every suite.
AC-4 carries the byte-level scaffold assertion that closes it.

### AC-2 — The JSON body-map help text is checked against that definition

The root banner's body-map clause is parsed and compared against the owned set,
slugified and in canonical order. The banner named `goal/acceptance_criteria` for
milestone against a three-section set, and the mismatch was the AC's red · commit
da7d134f9 · tests all green

The clause is parsed out of the real dispatcher's output rather than grepped, and
the parser fails loudly on a reworded anchor rather than passing over an empty
haystack.

### AC-4 — Approach is retired from the table and the doc surfaces follow

`Approach` left the owned set; the `aiwf-add` skill's required-sections table and
its milestone prose, the `aiwf-show` body-key table, the root help banner, and the
`entity-body-empty` rule's doc comment all followed — the last by dropping its
prose copy rather than updating it · commit 4cdbdaff6 · tests all green

What is pinned mechanically: the owned table itself, the `aiwf-add` required-sections
table, the `aiwf-show` row naming every owned slug, and the scaffold's rendered bytes
per kind — the last catching the display-form and blank-line edits that had been
passing every suite. What is not: the `aiwf-add` per-kind prose, and any key the
`aiwf-show` row carries beyond the owned set. Both are held at review.

The check's own fixtures used `## Approach` as their exemplar empty milestone
section and now use `## Goal`. The property each pins is unchanged.

## Decisions made during implementation

- **`Approach` is retired rather than kept and propagated upward.** The surfaces
  stating it disagreed, so this was a choice between them: the normative body-section
  statement and the prose template both omit it, and the template's `Context` and
  `Design notes` already hold what an implementation sketch would say. D-0065 records
  the follow-on question — whether `Context` takes the slot — as refused, since
  nothing replaced the section.
- **The check is not taught to fire on an absent heading.** Tracked as G-0571 with the
  measurement. A create-time refusal is the narrower option and belongs on its own
  evidence.
- **Three surfaces are left unpinned deliberately.** The `aiwf-add` per-kind prose,
  any `aiwf-show` key beyond the owned set, and the `## Approach` sections in existing
  bodies. The first two were guarded briefly and the guards were withdrawn: one banned
  the skill from naming legitimate template sections, the other constrained a table
  that legitimately carries extras. A mandate on prose costs every future edit; these
  are held at review instead.
- **The inert `len(RequiredSections) > 0` guard is deleted rather than annotated.**
  `EmptyRequiredSections` already returns nil for a kind with no set, so the guard
  gated nothing and allocated a discarded copy per entity per check run.

## Validation

`make ci` exit 0 — race suite, diff-scoped coverage gate, firing-fixture meta-gate,
profile-driven policy gates, and `aiwf doctor --self-check` at 29/29 steps including
`add milestone` and `check` against a temp repo.

`make lint` 0 issues. `go build ./...` clean. `aiwf check` 0 errors.

Behaviour confirmed against a binary built from this source, in a throwaway repo:
`aiwf add milestone` and `aiwf template milestone` scaffold `## Goal` and
`## Acceptance criteria`; a body carrying `## Approach` produces no finding, empty or
filled; `aiwf show --format=json` surfaces it as an ordinary author-added key.

Mutation-probed: a case-only edit to a section name, the loss of the blank line
between scaffolded headings, a reordered banner segment, a deleted banner segment, a
disabled `SectionSlug`, and a re-added `Approach` in either the owned table or the
`aiwf-add` table are each caught. The first two passed every suite before this
milestone.

## Deferrals

- **G-0571** — nothing enforces that a body carries its kind's required sections.
  Found while cancelling AC-3, and the reason the cancellation's original rationale
  did not hold. It carries the measurement and the two options.

## Reviewer notes

Reviewed by six independent fresh-context passes: four lenses over the change-set
(correctness, design, shipped consumer surfaces, blast radius) and two audits of the
retirement itself (residue sweep, coherence). Every pass returned request-changes;
every blocking finding is fixed or recorded below.

Two claims of the author's were measured false by review and corrected in place. The
retirement was first argued from "no authoring surface produces it", which the
`aiwf-add` skill falsified. AC-1's evidence claimed no surviving mutants from a probe
that only varied section names in ways that changed their slug; a case-only edit
survived everything, and AC-4's byte assertion is what closes it.

Declined, so a later reviewer meets a decision rather than a blank:

- **No guard over the `aiwf-add` per-kind prose or the `aiwf-show` extra keys.** See
  the decision above. Re-adding a retired section to either is caught at review only.
- **`docs/design/design-decisions.md` is a further surface stating the set** and is
  neither inventoried nor checked. It is Normative tier and its milestone row is
  correct; its "Bodies are not validated" line is stale independently of this work.
  Out of scope here.
- **The eight active milestone specs keep their `## Approach` sections.** Inert by
  design. No shipped surface asks for one now, so the population is closed.
- **`internal/skills/embedded/aiwf-add`'s "asserts presence, not structure" line** is
  the same overclaim G-0571 records, on a shipped surface. Pre-existing; folding it in
  would widen this milestone past its subject.
