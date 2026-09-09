---
id: M-0330
title: Check Unreleased against what shipped before the release tag
status: draft
parent: E-0091
tdd: required
---
## Goal

Make a release name everything it ships. An audit compares the shipped-surface
commits since the last release against what the `[Unreleased]` section names, and
the release tag's CI job fails when something is missing.

## Closes

<!-- Filled at milestone start, not while authoring this spec. -->

- (none)

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
