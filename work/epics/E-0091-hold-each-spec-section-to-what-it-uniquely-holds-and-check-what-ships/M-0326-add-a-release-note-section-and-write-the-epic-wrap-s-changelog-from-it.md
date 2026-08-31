---
id: M-0326
title: Add a Release note section and write the epic wrap's changelog from it
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: A section name a ritual writes resolves to a heading some artefact carries
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
resolves the backticked section names written in the ritual authoring tree
against the headings the shipped templates and the wrap artefact carry, so a
surface naming a section no artefact has is reported.

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

### AC-1 — A section name a ritual writes resolves to a heading some artefact carries

A check resolves each backticked `## Section` name mentioned in the ritual
authoring tree against the headings the shipped templates and the wrap
artefact's scaffold carry. A name matching none of them is reported, naming the
file and the line.

The evidence is a relationship between two artefacts rather than an assertion
about either one's prose, which is what D-0070 leaves available over a shipped
surface. What it catches is a name matching no artefact at all. It does not
catch renaming a heading that another template also carries, since the universe
is their union — the limit is stated in the policy's doc comment and belongs to
the section-ownership work G-0636 tracks.

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
does not relax a projection finding. Error severity also required fixtures
across four packages to seed a release note before promoting, and it put a
precondition on the milestone-to-done transition that the legal-workflow table
does not declare. Reverting to warning retired all of that; the fixtures
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
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-start-milestone/SKILL.md`
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/reviewer.md`
- `internal/skills/embedded/aiwf-check/SKILL.md`
- `internal/policies/` — AC-1's resolution check
- `internal/check/` — AC-3's finding rule, and its wiring into `check.Run`

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
- D-0082 — milestones contribute release notes, amending one of D-0031's rejected alternatives

---

## Release note

The shipped milestone-spec template now carries a `## Release note` section: the
user-visible delta of that milestone's work, written at its wrap by whoever did
it. The kernel's own scaffold still writes only `## Goal` and
`## Acceptance criteria`, which G-0656 records. `aiwfx-wrap-epic`
composes the epic's changelog entry from those notes rather than reconstructing
it from milestone titles and merge SHAs, and `aiwfx-wrap-milestone` puts the
note in front of the independent reviewer, since it is the input to the one
section that reaches the changelog verbatim.

`aiwf check` gains `milestone-done-empty-release-note` (warning): a non-archived
milestone at `done` with no `## Release note` an author wrote. It reports without
blocking — `aiwf check` and `promote` both gate on error severity — because at
error it would demand a section the kernel's own scaffold does not write. The archive gate keeps it
off milestones written before the section existed, once the sweep has moved
them.

A new policy check resolves the backticked section names written in the ritual
authoring tree against the headings the shipped templates and the wrap
artefact's scaffold carry, so a surface naming a section no artefact has is
reported. It resolves against the union of those artefacts rather than against
the one a mention names, so it catches a name that matches nothing and not a
rename of a heading another template still carries; per-target resolution is the
section-ownership work G-0636 tracks.

## Work log

### AC-1 — A section name a ritual writes resolves to a heading some artefact carries

`PolicyMilestoneSectionNameResolution` reports a shipped surface naming a section no template heading or wrap-artefact section carries · commit ff39bb5b2, largely rewritten in ec3d2a7d2, 1526a1124, b1c3b1ba6 and ed5937524 · tests: the `MilestoneSectionNameResolution`, `ReleaseNoteHeadingResolves` and `SectionSurfaces` cases in `internal/policies`

### AC-2 — Release note ships, and the epic wrap composes from it

Template ships `## Release note`; the milestone wrap reviews it at step 2 and the epic wrap composes its changelog entry from these notes · commit e2bb40351 · tests: AC-1's check, which went red on both rituals naming the section before the template carried it

### AC-3 — A done milestone with an empty Release note is reported

`milestone-done-empty-release-note` (warning) reports a done milestone whose release note nobody wrote; absence counts, so deleting the heading is not an escape · commit a6cbd3814, corrected in ec3d2a7d2, 1526a1124, dfa55e2bb and b1c3b1ba6 · tests: the `MilestoneDoneEmptyReleaseNote` cases in `internal/check`, including the one driving `check.Run`

## Decisions made during implementation

- **The release-note rule reports without blocking.** It ships at warning
  severity, so it reaches neither the push nor the `done` transition. Error was
  tried and reverted: it demands a section `entity.RequiredSections` does not
  write, so a milestone created through `aiwf add` cannot reach `done` and
  `--force` does not relax a projection finding. The consequence for the rest of
  this epic is that nothing mechanically stops an undescribed change from
  shipping — the `[Unreleased]` completeness check planned later is where that
  guarantee has to come from, not here.
- **D-0031's rejected alternative was revisited, and D-0082 records it.** That
  decision considered having each milestone contribute a note for the epic wrap
  to fold in, and rejected it as buying nothing the wrap artefact did not
  already give for free in the same sitting. The v0.34.0 evidence in `## Context`
  measures that premise false. D-0031's core holding survives untouched — the
  changelog still has one producer, the epic wrap — so D-0082 amends rather than
  supersedes it.
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
  as before the rule existed. The reason is the archive gate — every milestone
  already at `done` is archived, so none is in scope.
- Severity, measured rather than reasoned: `aiwf check` exits non-zero only via
  `check.HasErrors`, which matches error severity alone, and `promote` gates its
  projection findings the same way. A warning therefore reaches neither the push
  nor the transition.
- Statement coverage on both new files: every statement covered except one
  `//coverage:ignore`'d block, the `filepath.Rel` arm of the surface walk, which
  cannot fail for a path the walk itself produced. `go tool cover -func` reports
  93.8% on the function holding it and 100% on every other.
- Mutation probe: every mutant a review round found surviving is now killed —
  the kind guard, the heading-vs-empty flag, the severity, the two
  skip-condition arms, the span split, the scaffold-fence marker, and the status
  boundary between `done` and merely terminal. Each file was restored
  byte-identical to its pre-probe hash.

## Deferrals

- G-0656 — nothing records whether a rule keys to the kernel's scaffold section
  set or the ritual template's. Opened here: reverting this milestone's rule to
  warning sidesteps that collision rather than resolving it, and the same trap
  is live for four of the six entity kinds.

## Reviewer notes

Independent review rounds ran until one found no defect that was not already
pinned, recorded or tracked; each was a fresh-context pass over the full
change-set, the later ones sliced across the production units, the fixture blast
radius and the prose. Every round before the last returned REQUEST-CHANGES.

**The severity choice is the milestone's main trade-off.** The rule ships at
warning, so nothing blocks. Error was tried and reverted: it demands a section
`entity.RequiredSections` does not write, so a milestone created through
`aiwf add` could not reach `done`, and `--force` does not relax a projection
finding. It also required fixtures across four packages to seed a note before
promoting, and put a precondition on a transition the legal-workflow table
declares legal without declaring it there. The consequence to carry forward is
that nothing mechanically stops an undescribed change from shipping; that
guarantee has to come from the `[Unreleased]` completeness check G-0529 tracks.

**The surface set is derived, not declared.** A hand-maintained list fails
silently — add a ritual that names sections and nothing reports it. The
exemption list that a declared set would have needed was retired instead by
scaffolding the section it existed to excuse.

**Three limitations ship deliberately, each stated in the policy's doc comment
rather than implied away.** The universe is the union across every shipped
template, so a rename of a heading another template also carries is not caught.
A mention in a shipped tree this policy does not walk is out of reach. A backtick
span wrapped across a line is invisible to it. All three want a reference that
names which artefact it means, which is the section-ownership work G-0636 tracks.

**What the rounds kept finding is worth one sentence, because it is the
milestone's own subject.** The blocking defects moved from behaviour to claims
about behaviour: the code settled, and the prose describing it kept being wrong
in the section designed to travel into a changelog — each time because a
correction fixed the copy in front of it without searching for the others. The
tests were not immune: two cases named for distinct scaffold behaviours asserted
a condition that held under either reading, and statement coverage reported the
functions they covered at 100%.
