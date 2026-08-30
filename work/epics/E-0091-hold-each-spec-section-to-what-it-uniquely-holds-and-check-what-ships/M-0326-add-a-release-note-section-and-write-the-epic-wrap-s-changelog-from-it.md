---
id: M-0326
title: Add a Release note section and write the epic wrap's changelog from it
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: Every milestone-spec section a ritual names resolves to one the template ships
      status: met
      tdd_phase: done
    - id: AC-2
      title: The epic wrap composes its changelog entry from milestone Release notes
      status: met
      tdd_phase: done
    - id: AC-3
      title: A milestone reaching done with an empty Release note is reported
      status: met
      tdd_phase: done
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

Adding a section is not a single-file edit. Files across the ritual, template
and agent-card trees name `##` sections in backticked form — a mix of
milestone-spec, epic-spec and wrap-artefact names — and nothing today checks
that a name one surface uses matches a heading the artefact it names actually
carries.

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

`aiwf check` reports, at warning severity, a non-archived milestone at
`status: done` with no `## Release note` an author wrote — the section absent,
or present and carrying nothing but whitespace, headings, or the template's
guidance comment.

It reports and does not block. `aiwf check` exits non-zero on error severity
alone, and `promote` gates its projection findings the same way, so the rule
reaches neither the push nor the `done` transition.

Warning rather than error, decided against the measured cost of the alternative.
At error the rule demands a section the kernel's own scaffold does not write:
`entity.RequiredSections` for a milestone is Goal and Acceptance criteria, so a
milestone created through `aiwf add` could not reach `done` at all, and `--force`
does not relax a projection finding. Error severity also required six test
fixtures across four packages to seed a release note before promoting, and it
put a precondition on the milestone-to-done transition that the legal-workflow
table does not declare. Reverting to warning retired all of that; the fixtures
are back to their original shape, measured green.

Absence counts, rather than only an empty section that is present. Scoping to
present-and-empty would make deleting the heading an escape, and it would buy
nothing: the archive gate already spares every milestone written before the
section existed, measured at 281 archived and 0 live.

The rule is standalone: its own finding keyed to this section, not the first
consumer of `internal/entity/required_sections.go`. That file declares a
per-kind required set nothing enforces, and enforcing it would begin reporting
every entity missing a section its kind requires — G-0571 measures that at 119
findings over 60 live entities. The blast radius belongs to that gap, not to
this milestone.

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
- Enforcing the declared per-kind required-section set. AC-3 ships a standalone
  rule; folding it into that machinery, or recording why it stays separate, is
  an obligation carried by whoever closes G-0571 and is written there.

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

## Release note

Milestone specs now carry a `## Release note` section: the user-visible delta of
that milestone's work, written at its wrap by whoever did it. `aiwfx-wrap-epic`
composes the epic's changelog entry from those notes rather than reconstructing
it from milestone titles and merge SHAs, and `aiwfx-wrap-milestone` puts the
note in front of the independent reviewer, since it is the one spec section that
travels verbatim into the changelog.

`aiwf check` gains `milestone-done-empty-release-note` (warning): a non-archived
milestone at `done` with no `## Release note` an author wrote. It reports without
blocking — `aiwf check` and `promote` both gate on error severity — because at
error it would demand a section the kernel's own scaffold does not write. Every
milestone already at `done` is archived, so the rule reports on none of them.

A new policy check resolves every milestone-spec and wrap-artefact section name
the shipped rituals, agent cards and templates mention against the headings
those artefacts actually carry, so renaming a section on either side is caught
rather than left to drift.

## Work log

### AC-1 — Section names resolve against the artefacts that carry them

`PolicyMilestoneSectionNameResolution` reports a shipped surface naming a section no template heading or wrap-artefact section carries · commit ff39bb5b2 · tests 15/15

### AC-2 — Release note ships, and the epic wrap composes from it

Template ships `## Release note`; the milestone wrap reviews it at step 2 and the epic wrap composes its changelog entry from these notes · commit e2bb40351 · tests 15/15 via AC-1's check, which went red on all three surfaces before the section existed

### AC-3 — A done milestone with an empty Release note is reported

`milestone-done-empty-release-note` (warning) reports a done milestone whose release note nobody wrote; absence counts, so deleting the heading is not an escape · commit a6cbd3814, corrected in ec3d2a7d2 and the round that followed · tests 13/13

## Decisions made during implementation

- **The release-note rule reports without blocking.** It ships at warning
  severity, so it reaches neither the push nor the `done` transition. Error was
  tried and reverted: it demands a section `entity.RequiredSections` does not
  write, so a milestone created through `aiwf add` cannot reach `done` and
  `--force` does not relax a projection finding. The consequence for the rest of
  this epic is that nothing mechanically stops an undescribed change from
  shipping — the `[Unreleased]` completeness check planned later is where that
  guarantee has to come from, not here.
- **The epic wrap reads each milestone's `## Release note` directly.** The epic
  spec left open whether the wrap reads the notes or the notes accumulate
  somewhere it copies from. Reading directly needs no new artefact and no second
  place for the text to drift; the cost is that the wrap must open each wrapped
  milestone's spec, which it already does for `## Milestones delivered`.

## Validation

Run on the milestone branch in the devcontainer (Linux; `go test` runs unwrapped
there, so no signing wrapper is involved).

- `AIWF_COVERAGE_BASE=main make ci` — exit 0. Race suite, the diff-scoped
  coverage audit against `main`, the profile-driven gates, and the 29-step
  `aiwf doctor --self-check`.
- `aiwf check` — 0 errors, 2 warnings, both predating this milestone:
  `epic-active-no-drafted-milestones` on the parent epic, which follows from
  allocating one milestone at a time, and
  `provenance-untrailered-scope-undefined`, which follows from the branch having
  no upstream.
- Live-tree behaviour of the new rule, measured with a binary built from this
  branch rather than the one on PATH: `aiwf check` reports the same two warnings
  as before the rule existed. Two independent reasons it reports nothing: 281
  milestones are `done` and none carries the section, and 0 milestones are
  `done` and not archived.
- Severity, measured rather than reasoned: `aiwf check` exits non-zero only via
  `check.HasErrors`, which matches error severity alone, and `promote` gates its
  projection findings the same way. A warning therefore reaches neither the push
  nor the transition.
- Statement coverage on both new files: 100%.
- Mutation probe, final round: 7 mutants across the two units, all killed, each
  file restored byte-identical to its pre-probe hash. Every one of the seven was
  a mutant that survived an earlier round — the kind guard, the heading-vs-empty
  flag, the severity, the two skip-condition arms, the span split, and the
  scaffold-fence marker.

## Deferrals

- (none)

## Reviewer notes

Two review rounds ran, each a fresh-context pass over the full change-set; the
second was sliced across the production units and the fixture blast radius. Both
rounds returned REQUEST-CHANGES, and the findings that mattered were claims this
spec made that measurement contradicted.

The first round found that the rule did not do what three surfaces said it did.
It was written as a warning while its code comment, this spec's AC-3 body, and
this milestone's own `## Release note` all said it refused the `done` promote —
which `promote` gates on error severity, so it never did. No test covered the
claim. The same round found the scope argument void: the 281 `done` milestones
cited to justify reporting only a present-and-empty section are all archived,
and the rule already skips archived, so the narrower scope bought nothing and
left deleting the heading as an escape.

The second round found the correction had introduced its own contradictions. A
release note containing only a sub-heading escaped the rule while both the code
comment and the shipped `aiwf-check` skill said headings count as unwritten; the
flag deciding it passed the whole suite in either position. The guard meant to
stop the wrap-artefact exemption list from growing was circular — it accepted an
entry on the strength of a backticked mention, which is exactly what creates the
need to exempt one. The operator message and hint spelled the section name
independently of the constant the rule reads, so a coherent rename left them
lying. The span splitter fired on any slash, so a heading named "Client/server
split" would have become two names that resolve nowhere. And the claim that the
derived surface scan needs no exemptions held only inside one chosen directory:
the `aiwf-add` skill names those sections at length and is not walked.

The escalation to error was reverted on the evidence those rounds produced. It
demanded a section the kernel's own scaffold does not write, so a milestone
created through `aiwf add` could not reach `done`; it required six fixtures
across four packages to seed a note before promoting; and it put an undeclared
precondition on a transition the legal-workflow table declares legal. All three
retired with the severity. The exemption list retired too, by scaffolding the
section it existed to excuse.

What is deliberately left: a rename can still strand a mention in a shipped tree
this policy does not walk, and a backtick span wrapped across a line is invisible
to it. Both are stated in the policy's doc comment rather than implied away, and
resolving them needs a reference that names which artefact it means — the
section-ownership work G-0636 tracks.
