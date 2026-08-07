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
      title: Approach leaves the milestone required set with no grandfather allowlist
      status: open
---

## Goal

Give the per-kind body-section set one definition, and make the rule that names it
report a section that is absent — not only one that is present and empty.

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
So the requirement holds only over sections that happen to be present, and membership
is unenforced.

Measured in a scratch consumer repo: `aiwf add milestone` writes a body missing a
section the kernel declares required, and `aiwf check` reports nothing on that axis.

## Approach

Give the set one owner and have the other surfaces read it rather than restate it.
The scaffold is the surface that must derive — a literal beside the table is what let
the two drift in the first place. The help text is checked rather than generated,
since it is prose describing a projection rather than a copy of the set.

Then give the rule the absent case. It routes through the helper
`internal/verb/add.go` and the `entity-body-empty` rule already share, so the verb-time
refusal and the check-time finding cannot disagree about what "missing" means — the
reason that helper was extracted.

Membership is decided last and on evidence. `Approach` entered the set by
transcription rather than by decision: M-0066 introduced the rule and enumerated the
load-bearing sections from its own body, whose first three headings are `Goal`,
`Approach`, `Acceptance criteria`. G-0058, the gap that motivated the rule, asks only
that AC prose not ship blank and never names the section at all. The shipped
milestone template has carried no `## Approach` in any revision of its history — its
second section is `## Context`.

So the requirement names a section no authoring surface has ever produced, and
because the rule reports only present-but-empty, it has never caused an `Approach`
section to exist that would not have existed anyway. Removing it does not relax a
standard; it withdraws one that was never applied. The section stays permitted, and
milestones with something to say about method keep saying it. Whether `Context`
should take the vacated slot is a separate question, tracked as D-0065.

Ordering matters within the milestone: the absent-section finding must not land before
`Approach` leaves the set, or every existing milestone flags in between.

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

### AC-4 — Approach leaves the milestone required set with no grandfather allowlist

The milestone entry in the owned set does not name `Approach`. No allowlist,
exemption ledger, or grandfather entry is introduced by this milestone. Every
non-archived, non-terminal milestone carries every section the set names, both
before the removal and after, so the change is finding-neutral for the kind. What
the wider tree does once absence is a finding is AC-3's scoping question, not this
criterion's.

The `aiwf-show` skill's body-key table is the sixth surface stating the set, and it
follows in the same change. Its milestone row drops `approach` when the set does.
Its gap row names `what_s_missing` — the slug `SectionSlug` derives from
`What's missing` — rather than `whats_missing`, which no `aiwf show` envelope
carries. A test pins the table's rows against the owned set, so this surface cannot
drift silently the way the help text did.

## Constraints

- No new chokepoint. The absent case flows through `EmptyRequiredSections` rather
  than a second helper beside it.
- Layering direction is already policed; wherever the owned set lands, the existing
  layering rule decides the direction, not convenience.
- No grandfather allowlist survives this milestone.

## Design notes

Where the set lives is open. `internal/check` reads it today, but `entity.BodyTemplate`
must derive from it and `entity` sits below `check`, so the table likely moves down
rather than the dependency moving up. Settled against the layering rule during AC-1,
not guessed here.

Whether the absent case is a new finding code or a subcode of the existing one is an
implementation choice for AC-3; the acceptance criterion pins the observable finding,
not its code.

## Surfaces touched

`internal/check/entity_body.go`, `internal/entity/serialize.go`,
`internal/verb/add.go`, `internal/cli/root.go`, `internal/policies/`.

## Out of scope

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
updated · commit 5227fef7f · tests all green, 0 surviving mutants

### AC-2 — The JSON body-map help text is checked against that definition

The root banner's body-map clause is parsed and compared against the owned set,
slugified and in canonical order; it was naming `goal/acceptance_criteria` for
milestone while the set carries `Approach` · commit da7d134f9 · tests all green,
0 surviving mutants

## Decisions made during implementation

## Validation

## Deferrals

## Reviewer notes
