---
id: M-0330
title: Check Unreleased against what shipped before the release tag
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: The audit reports a shipped-surface delta nothing under Unreleased cites
      status: met
      tdd_phase: done
    - id: AC-2
      title: A shipped-surface commit with no entity trailer is reported, not skipped
      status: met
      tdd_phase: done
    - id: AC-3
      title: The base release is the newest tag reachable from HEAD
      status: met
      tdd_phase: done
    - id: AC-4
      title: The release tag workflow invokes the audit and the target it names exists
      status: open
      tdd_phase: green
---
## Goal

Make a release name everything it ships. An audit compares the shipped-surface
commits since the last release against what the `[Unreleased]` section names, and
the release tag's CI job fails when something is missing.

## Closes

- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never
  checked.

The gap's direction names two properties and this milestone delivers one of
them over one surface class: a shipped delta named under `[Unreleased]` before
a release, enumerated over the embedded trees. That is the class the v0.34.0
omissions came from, and the gap records that its own named-surface list —
finding codes, verbs, config keys, exit codes — would not have reached them.
The named-surface half stays undone, and it is the half that would have caught
the earlier thin entry, so it is a residual rather than a subsumed alternative.
It takes its own gap at wrap.

The per-epic citation property falls out of the same comparison wherever an
epic's work touched a shipped surface. An epic that shipped only compiled
behaviour is outside what this milestone sees, and rides the same residual.

## Context

The `[Unreleased]` section is written at two moments: a patch's own wrap, and an
epic's wrap. Nothing verifies the result. The tag workflow fires on a pushed
`v*` tag and confirms a `## [X.Y.Z]` heading exists, which a stub satisfies.

Measured 2026-09-09 on this branch: 18 non-merge commits have touched
`internal/skills/` since v0.34.0, and every one carries an `aiwf-entity` trailer.
Rolling each milestone up to its parent epic leaves five entities owing an entry,
of which `[Unreleased]` names three. The two it does not name are E-0091, whose
milestone deltas await its wrap entry, and G-0659, a patch that changed a shipped
ritual and wrote no line although the patch ritual mandates one with no skip.

That trailer is what makes the property computable: a shipped-surface change
already names the entity that owns it. It is guaranteed only for the ritual
skill files under `internal/skills/embedded-rituals`, though — not for the verb
skills, the templates, the agent cards or the guidance fragment. Elsewhere it is
habit, and the habit is one release cycle old: across the three ranges that
already shipped, the untrailered share of shipped-surface commits ran 20 of 20
into v0.32.0, then 38 of 52, then 2 of 11, against 0 of 18 on the current range.
An audit that trusts the trailer has to say when it is absent.

Backtested over those same three ranges, the comparison reports two uncited
entities into v0.33.0 and none into the other two. One of the two is documented:
`[0.33.0]` carries an entry for G-0635's change that names no id. So what the
audit proves is that nothing under the section cites the entity, which is a rule
it imposes rather than an omission it observes — an entry describing a change
while naming no id is indistinguishable from no entry at all. D-0087 settles
what each finding then does: the uncited entity fails the release, the
untrailered commit is reported and does not.

The backlog is two entities and both clear before the audit can block: E-0091's
at its own wrap, G-0659's as a changelog line this milestone writes.

## Acceptance criteria

### AC-1 — The audit reports a shipped-surface delta nothing under Unreleased cites

The report names the entity nothing cites and at least one commit behind it, and
it fails the release. Naming the entity is what the test asserts, not the exit
code alone: an unreadable changelog and an unresolvable base ref both fail on
this path already, so an exit-code assertion passes with the comparison absent.

A milestone rolls up to its parent epic, and the epic is what must be cited — a
milestone's user-visible delta lands in its epic's entry, never its own. The
rollup resolves through the tree loader, which reaches archived entities. A path
scan does not, and the failure is silent rather than loud: an archived
milestone's id survives the rollup and is reported as an uncited entity in its
own right. A composite trailer value resolves to its milestone before the
rollup, and ids compare canonicalized, since a narrower legacy width names the
same entity.

Merge commits are excluded. Their file lists carry the merged branch's changes,
already attributed to the commits that made them, so counting them attributes
one delta twice.

Evidence: a fixture repo whose range carries one shipped-surface commit under an
entity the section does not cite, asserted to report that entity; the same range
with the entity cited, asserted to report nothing. The rollup runs against an
archived milestone, which is the reading a path scan gets wrong; the
narrow-width reading gets a range that turns on nothing else.

What turns that report into a failed release is the shared policy harness, which
fails one test per violation, and the wiring that reaches the harness from the
release tag. Neither is asserted here: the harness is pinned by every policy in
the suite, and the wiring is AC-4's.

### AC-2 — A shipped-surface commit with no entity trailer is reported, not skipped

An untrailered shipped-surface commit is reported by subject, at a severity that
does not fail the release. Skipping it is the failure mode the audit is most
likely to have: the comparison is driven by trailers, so a commit carrying none
contributes to neither side and passes silently.

The trailer is guaranteed on one tree only. The provenance backstop covers the
ritual skill files; the verb skills, the templates, the agent cards and the
guidance fragment carry a trailer by habit. An audit that trusts the trailer
everywhere under-reports exactly where the guarantee stops, and says nothing
while it does.

It reports rather than blocks because it establishes nothing about the
changelog: the audit cannot attribute the commit, so it cannot say whether an
entry covers it, and the repair a failing release would imply — rewriting a
landed commit's trailers — is not available. D-0087 carries the argument.

Evidence: a range carrying one shipped-surface commit with no entity trailer and
nothing else uncited, asserted to report that commit and to produce no
release-failing finding. Both halves are asserted, because each fails a
different wrong implementation: one that treats an absent trailer as nothing to
attribute drops the report, and one that routes it to the blocking half instead.
The second assertion runs through the release-gating entry point rather than the
renderer beneath it, since a leak into the blocking half leaves the renderer
untouched. Measured on the current range the tree offers no such commit, so the
fixture builds one rather than reading history.

### AC-3 — The base release is the newest tag reachable from HEAD

The range the audit reads starts at the newest release tag reachable from the
commit under test. A tag on a branch that commit cannot reach is not its base,
and choosing it compares the change against a release that never contained it.

This is where a branch and trunk part. The audit runs on a branch in practice —
the local gate runs before the merge — while on trunk the newest tag and the
newest reachable tag are usually the same commit. A trunk-only test therefore
passes against a resolution that reads the tag list and sorts it.

Evidence: one fixture repo carrying a tag on a branch the commit under test
cannot reach, asserted to resolve to the reachable tag instead. A repo with no
tag at all resolves to the root commit rather than failing, since a first
release has no predecessor.

### AC-4 — The release tag workflow invokes the audit and the target it names exists

The workflow that fires on a release tag runs the audit, and the target it names
exists. Both halves, because either alone is satisfied by a broken wiring: a
step naming a target the Makefile does not define fails only when someone cuts a
release, and a defined target nothing invokes is a command that never runs.

The claim is scoped to the wiring, not to the outcome. Whether the job fails a
release whose notes are incomplete cannot be asserted without running the
workflow, which the test suite does not reach. What it can assert is that the
invocation resolves, so renaming either side reports.

Three links, not two: the workflow step names a target, the target runs a test
by name, and that test is the audit's release-gate entry point. A chain asserted
only at its ends stays green while its middle names a test that no longer
exists — a break that surfaces when someone cuts a release and nowhere earlier.

Evidence: the recipe resolved by running make rather than by reading the
Makefile, so renaming an intermediate target keeps it green and only the audit
dropping out turns it red; the workflow's step asserted to invoke that target;
and the resolved recipe asserted to name the audit's entry-point test.

## Constraints

- Scope is this repo. The embedded trees exist only in aiwf's own source, so a
  consumer running this audit would check nothing. It is a repo invariant, not a
  shipped verb, and no consumer-facing surface changes.
- No new top-level verb, no new skill, no completion wiring.
- The audit runs at the release boundary, not at push. A milestone delta awaiting
  its epic's wrap entry is correctly absent until a release intervenes, which is
  why asking at push time would need an in-flight-epic exemption and asking at the
  tag needs none.
- Attribution follows the rule already documented: a milestone's delta belongs to
  its parent epic's entry, not its own.

## Design notes

- The tag workflow already fires on `v*` tags. The audit joins that workflow
  rather than adding one.
- `make comment-history-audit` is the shape precedent: a policy that also carries
  a focused target.
- D-0031 fixed the changelog category set and G-0613 questions it. Not settled
  here — that is a decision amendment, not a check.
- The two findings carry different severities, settled in D-0087. An uncited
  entity is a proven violation of a stated rule; an untrailered commit is a hole
  in the audit's own evidence.
- The rollup from milestone to parent epic goes through `tree.Load`, never a
  path scan. The loader resolves across active and archive; a glob over
  `work/epics/*/` does not, and an archived milestone then survives the rollup
  and reports as an uncited entity of its own.

## Surfaces touched

- `internal/policies/`
- `Makefile`
- `.github/workflows/changelog-check.yml`
- `CLAUDE.md` (release process)

## Out of scope

- The named-surface half of G-0529's direction — finding codes, verb names,
  config keys, exit codes. Those need list comparison rather than trailer
  reading, and the case that failed twice is the embedded trees.
- A changelog check for consumer repos. Different surfaces, no evidence yet.
- G-0613's category set.

## Dependencies

- None.

## Coverage notes

- (none yet)

## References

- G-0529 — CHANGELOG completeness rests on recall at epic wrap and is never checked
- G-0613 — the wrap changelog category set omits Removed, which practice uses
- D-0031 — changelog entries are copied from wrap.md, not independently authored

## Release note

## Decisions made during implementation

- D-0087 — the changelog audit blocks an uncited entity and reports an
  untrailered commit

## Validation

## Deferrals

## Reviewer notes

- Content introduced only by a merge resolution is invisible to this audit.
  Measured: `git log --name-only` emits no file list for a merge without an
  explicit `--diff-merges`, and no `log.diffMerges` setting changes that, so
  such content belongs to no non-merge commit and is attributed to nothing. It
  is the same blind spot G-0602 tracks for the shipped-ritual provenance gate,
  and closing it there closes it here.
