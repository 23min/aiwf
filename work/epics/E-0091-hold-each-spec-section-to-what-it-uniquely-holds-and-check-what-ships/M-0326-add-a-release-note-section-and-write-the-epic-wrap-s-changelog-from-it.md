---
id: M-0326
title: Add a Release note section and write the epic wrap's changelog from it
status: draft
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: Every milestone-spec section a ritual names resolves to one the template ships
      status: open
    - id: AC-2
      title: The epic wrap composes its changelog entry from milestone Release notes
      status: open
    - id: AC-3
      title: A milestone reaching done with an empty Release note is reported
      status: open
---

## Goal

Give the milestone spec a `## Release note` — the user-visible delta of that
milestone's work — and make the epic wrap's changelog entry read from those
notes instead of from milestone titles and merge SHAs. Ship a check that
resolves every milestone-spec section name a shipped surface mentions against
the headings the template actually carries, so the new section stays consistent
across the surfaces that name it.

## Context

`aiwfx-wrap-epic` authors `## Changelog entry` by hand at step 1 of its wrap
artefact and copies it verbatim into `CHANGELOG.md` at step 6. No code produces
it. Its inputs are `## Milestones delivered` — titles and merge SHAs — and the
author's memory of the epic, because no milestone spec records the user-visible
delta of its own work. The entry is therefore reconstructed at the end from the
thinnest available evidence.

Cutting v0.34.0 showed what that costs: three changes reached the release
undocumented, two of them on `docs(` commits — a prefix that means "nothing
user-visible" in most repos and the opposite here, since guidance and rituals
ship as product (G-0529).

Adding a section is not a single-file edit. The template carries sixteen `##`
sections, and ten files across the ritual, template and agent-card trees name
some `##` section in backticked form — a mix of milestone-spec and epic-spec
names, which is why telling the two apart is an edge AC-1 has to handle.
Nothing today checks that a name one surface uses matches a heading the template
ships.

## Acceptance criteria

### AC-1 — Every milestone-spec section a ritual names resolves to one the template ships

A check resolves each milestone-spec section name mentioned in the shipped
ritual, agent-card and template trees against the heading set
`templates/milestone-spec.md` carries. A name matching no heading is reported,
naming the file and the name.

The evidence is a relationship between two artefacts rather than an assertion
about either one's prose, which is what D-0070 leaves available over a shipped
surface: renaming a heading in the template reddens it, and so does a ritual
naming a section that does not exist.

One edge to settle in implementation: epic-spec section names live in the same
trees, so the check must not resolve an epic-spec name against the milestone
template. Whether they are told apart by naming file, by an explicit inventory,
or by resolving against either template is an implementation choice.

### AC-2 — The epic wrap composes its changelog entry from milestone Release notes

`aiwfx-wrap-epic` step 1 composes `## Changelog entry` from the wrapped
milestones' `## Release note` sections. `aiwfx-wrap-milestone` fills that note
as part of its wrap. The template ships the section with a comment stating what
it holds — this milestone's user-visible delta — and what it does not: neither
the Work log's commit index nor the epic-level summary.

Both section names resolve under AC-1's check, so a rename on either side goes
red. What the prose *instructs* cannot be pinned beyond that (D-0070), and its
correctness is held at review. If implementation finds nothing further to pin,
this criterion is recorded as an observation naming the ritual step and the
section it reads — never restated into a different, checkable claim standing in
for it.

### AC-3 — A milestone reaching done with an empty Release note is reported

`aiwf check` reports a milestone whose status is `done` and whose
`## Release note` is absent or empty.

Two things to settle in implementation:

- *Blast radius.* Every milestone already `done` lacks the section. The rule
  lands at warning severity against a measured baseline and escalates only once
  that baseline is clean.
- *Overlap.* `internal/entity/required_sections.go` already declares a per-kind
  required set that nothing enforces — G-0571, scheduled to a later milestone in
  this epic. Decide whether this rule is that machinery's first consumer or a
  standalone rule, and if standalone, say why.

## Constraints

- Every edit to a `SKILL.md` under `internal/skills/embedded-rituals/` rides a
  commit carrying an `aiwf-entity` trailer naming this milestone, or
  `skill-edit-provenance-backstop` fails the gate.
- No prose- or heading-presence assertion over a shipped surface. D-0070 retires
  the class across the template, ritual, agent-card and guidance trees; what
  survives is the relationship check AC-1 builds.
- Shipped surfaces cite no real entity id, filesystem path or lifecycle status.
  `skill-body-id` fires at error severity on a real id and on a placeholder at
  any width but canonical.
- `## Work log` stays. Retiring it depends on `aiwf history` seeing
  entity-trailered commits and belongs to a later milestone; the two sections
  hold different facts and coexist until then.

## Design notes

- The changelog entry is authored, not generated — D-0031 settles that it is
  copied from the wrap artefact rather than independently written. This
  milestone changes what the author is told to read, not a code path.

## Surfaces touched

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/templates/milestone-spec.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-milestone/SKILL.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`
- `internal/policies/` — AC-1's resolution check
- `internal/check/` — AC-3's finding rule

## Out of scope

- Retiring `## Work log`. A later milestone, gated on the history projection.
- Naming an owner for each section's population-timing rule (G-0636). AC-1
  checks that a section *name* resolves, which is narrower than the ownership
  that gap asks for and does not settle it.
- Checking that `[Unreleased]` names everything that shipped (G-0529). This
  milestone supplies the input; the check is a separate milestone.
- Widening the changelog category set. That is a decision revisit tracked in
  G-0613.

## Dependencies

- None. No other milestone in this epic gates this one.

## References

- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never checked
- G-0636 — milestone-spec section rules are restated across five surfaces with no owner
- G-0571 — nothing enforces that an entity body carries its kind's required sections
- G-0613 — the wrap changelog category set omits Removed, which practice uses
- D-0070 — prose-content assertions over shipped surfaces are retired
- D-0031 — CHANGELOG entries are copied from the wrap artefact, not independently authored

---

## Work log

## Decisions made during implementation

- (none)

## Validation

## Deferrals

- (none)

## Reviewer notes

- (none)
