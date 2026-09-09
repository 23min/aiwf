---
id: M-0330
title: Check Unreleased against what shipped before the release tag
status: in_progress
parent: E-0091
tdd: required
acs:
    - id: AC-1
      title: The audit reports a shipped-surface delta nothing under Unreleased cites
      status: open
    - id: AC-2
      title: A shipped-surface commit with no entity trailer is reported, not skipped
      status: open
    - id: AC-3
      title: The base release is the newest tag reachable from HEAD
      status: open
    - id: AC-4
      title: The release tag workflow invokes the audit and the target it names exists
      status: open
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
skills, the templates, the agent cards or the guidance fragment. Every recent
commit carries one by habit rather than by rule, so an audit that trusts it has
to say when it is absent.

The backlog is two entities and both clear before the audit can block: E-0091's
at its own wrap, G-0659's as a changelog line this milestone writes.

## Acceptance criteria

### AC-1 — The audit reports a shipped-surface delta nothing under Unreleased cites

The report names the entity that owes the entry and at least one commit behind
it. Naming it is what the test asserts, not the exit code alone: an unreadable
changelog and an unresolvable base ref both fail on this path already, so an
exit-code assertion passes with the comparison absent.

A milestone rolls up to its parent epic, and the epic is what must be named — a
milestone's user-visible delta lands in its epic's entry, never its own. A
composite trailer value resolves to its milestone before that rollup. An id
written at a legacy narrow width names the same entity as the canonical width,
so both sides are canonicalized rather than string-matched.

Merge commits are excluded. Their file lists carry the merged branch's changes,
already attributed to the commits that made them, so counting them attributes
one delta twice.

Evidence: a fixture repo whose range carries one shipped-surface commit under an
entity the section does not name, asserted to report that entity and exit
non-zero; the same range with the entity named, asserted to report nothing at
exit zero. The rollup and the narrow-width readings each get a range that turns
on nothing else.

### AC-2 — A shipped-surface commit with no entity trailer is reported, not skipped

An untrailered shipped-surface commit is reported by subject and counts toward
the non-zero exit. Skipping it is the failure mode the audit is most likely to
have: the comparison is driven by trailers, so a commit carrying none
contributes to neither side and passes silently.

The trailer is guaranteed on one tree only. The provenance backstop covers the
ritual skill files; the verb skills, the templates, the agent cards and the
guidance fragment carry a trailer by habit. An audit that trusts the trailer
everywhere under-reports exactly where the guarantee stops, and says nothing
while it does.

Evidence: a range carrying one shipped-surface commit with no entity trailer and
nothing else owing an entry, asserted to report that commit and exit non-zero —
which fails against an implementation that treats an absent trailer as nothing
to attribute. Measured today the tree offers no such commit, so the fixture
builds one rather than reading history.

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

Evidence: the workflow file parsed, the step's command extracted, and the target
it names looked up in the Makefile — asserted in both directions, since a test
that only reads the workflow passes after the target is deleted.

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

## Validation

## Deferrals

## Reviewer notes
