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
      status: open
      tdd_phase: done
---

## Goal

Give the per-kind body-section set one definition that every other surface derives
from or is checked against, and retire the one entry that no surface ever produced.

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

Membership is not left to nothing, though: `aiwf add` scaffolds every section the set
names, so an entity created through the verb carries them all and the emptiness rule
then has something to police. Absence arises only where a body bypasses the scaffold
— `--body-file` at create time, or hand-authoring. That division is why this milestone
repairs the definition rather than the rule.

Measured in a scratch consumer repo: `aiwf add milestone` writes a body missing a
section the kernel declares required, and `aiwf check` reports nothing on that axis.

## Approach

Give the set one owner and have the other surfaces read it rather than restate it.
The scaffold is the surface that must derive — a literal beside the table is what let
the two drift in the first place. The help text is checked rather than generated,
since it is prose describing a projection rather than a copy of the set.

The rule itself is left alone. Firing on an absent heading would raise 118 findings
against the tree as it stands — 61 gaps and 57
decisions, both born-complete kinds the rule scores at error severity — against
entities nobody was worried about, to close a leak that only `--body-file` and
hand-authoring open. The scaffold already supplies membership on every path that
goes through the verb. Whether that remaining leak is worth a create-time gate is a
question on its own evidence, not a rider on this one.

Membership is decided last and on evidence. `Approach` entered the set by
transcription rather than by decision: M-0066 introduced the rule and enumerated the
load-bearing sections from its own body, which carried `Goal`, `Approach`,
`Acceptance criteria`. G-0058, the gap that motivated the rule, asks only that AC
prose not ship blank and never names the section at all.

The surfaces that state the set disagree about it, so retiring it is a choice between
them rather than the removal of something uncontested. `internal/skills/embedded/aiwf-add`
names `Approach` required and describes it as the implementation sketch. Against that:
`docs/design/design-decisions.md` is the normative statement of what a milestone body
carries, predates the rule, and omits the section; the prose milestone template that
the planning ritual actually fills has never carried it in any revision of its
history; and that template's `Context`, `Design notes` and `Constraints` already hold
what an implementation sketch would say. The disagreement resolves downward — the
section leaves the set, and the `aiwf-add` skill follows.

The section stays permitted, and milestones with something to say about method keep
saying it; the bodies that already carry one are unaffected. Whether `Context` should
take the vacated slot is a separate question, tracked as D-0065.

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

Every remaining surface that states the set follows in the same change, and a test
pins each against the owned definition so none can drift silently the way the help
text did:

- `internal/skills/embedded/aiwf-add` — a required-body-sections table and a prose
  paragraph per kind. Both name `Approach`; both are shipped to consumer repos and
  are what an authoring assistant reads, so leaving either would reproduce the drift
  this milestone exists to end.
- `internal/skills/embedded/aiwf-show` — the body-key table. Its milestone row drops
  `approach`. Its gap row names `what_s_missing`, the slug `SectionSlug` derives from
  `What's missing`, rather than `whats_missing`, which no `aiwf show` envelope
  carries.
- `internal/check/entity_body.go` — the package doc comment restates all six kinds in
  prose, beside the literal this milestone already removed.

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

`internal/check/entity_body.go`, `internal/entity/serialize.go`,
`internal/entity/required_sections.go`, `internal/cli/root.go`,
`internal/skills/embedded/aiwf-show/SKILL.md`, `internal/policies/`.

## Out of scope

- Making the check fire on an absent section. `aiwf history M-0305/AC-3` carries the
  measurement that settled it.
- A create-time gate refusing an `aiwf add --body-file` body that omits a section the
  set names. It is the only remaining path to an absent section, but it is a new
  requirement and belongs on its own evidence.
- Whether `Context` should take the slot `Approach` vacates (D-0065).
- The shipped prose templates, which M-0306 reconciles against the owned set.
- The always-on guidance's scaffold instruction, which M-0307 routes.
- Whether the milestone template's duplicating sections should exist at all (G-0530).

## Dependencies

None. First milestone of E-0081.

## References

- G-0482 — `Approach` exists on no shipped surface
- E-0081 — parent epic, carrying the six-surface inventory

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
slugified and in canonical order; it was naming `goal/acceptance_criteria` for
milestone while the set carries `Approach` · commit da7d134f9 · tests all green,
0 surviving mutants

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
