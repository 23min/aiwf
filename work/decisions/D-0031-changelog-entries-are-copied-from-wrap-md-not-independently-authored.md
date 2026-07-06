---
id: D-0031
title: CHANGELOG entries are copied from wrap.md, not independently authored
status: proposed
relates_to:
    - G-0368
---
> **Date:** 2026-07-06 · **Decided by:** human/peter

## Question

Where should the prose for an epic's `CHANGELOG.md` `[Unreleased]` entry be
authored, and how should it relate to `wrap.md`'s own `## Summary` section —
both written in the same wrap sitting, describing the same epic, for
different readers?

## Decision

`wrap.md` is the single point of authorship for what to tell people about an
epic. It carries two adjacent, audience-distinct sections:

- `## Summary` (existing) — internal, framed against the epic spec's own goal
  language, for a reader following the project's planning state.
- `## Changelog entry` (new) — written for a release-notes reader who has
  never seen the epic spec, using the Keep-a-Changelog heading shape already
  in use today (`### Added — E-NN: <summary>` / `### Changed` / `### Fixed`),
  with an optional bullet per milestone when more than one shipped a distinct
  user-visible change. Sits directly beneath `## Milestones delivered` in the
  `wrap.md` template.

`aiwfx-wrap-epic` step 7 changes from freely re-authoring a CHANGELOG
paragraph to copying the `## Changelog entry` section verbatim into
`CHANGELOG.md` under `[Unreleased]`. One authoring act, two placements.

`aiwfx-wrap-milestone` stays changelog-free — only epic wrap ever writes to
`CHANGELOG.md`. The milestone wrap commit's one-line summary
(`feat(<scope>): <one-line summary> (M-NNNN)`) is VCS metadata for engineers,
not upstream material for this funnel. AC-level detail never surfaces in
`CHANGELOG.md` at any granularity.

## Reasoning

The two-section split treats "what happened" (Summary) and "what to tell a
release-notes reader" (Changelog entry) as genuinely different writing tasks
with different audiences, rather than collapsing them into one blurb reused
for both, or independently authoring both from the same mental model — the
latter is what creates drift risk: a wrap author writing a CHANGELOG
paragraph from memory, independently of the wrap.md summary they just wrote,
can silently drop a milestone from one without the other catching it.

Alternatives considered:

- **Push a changelog note down into `aiwfx-wrap-milestone`**, so each
  milestone contributes its own line for the epic wrap to fold in. Rejected:
  makes milestone wrap a second CHANGELOG-bound producer, breaking the
  single-producer pattern already established for `wf-patch`, without buying
  anything the wrap.md funnel doesn't already give for free in the same
  sitting.
- **A structural check that every `wrap.md`-listed milestone id appears in
  the CHANGELOG entry.** Rejected as the primary fix: the section-adjacency
  in `wrap.md` (Changelog entry sits directly below Milestones delivered)
  already puts the list in front of the author while they write; a
  mechanical check would add enforcement for a risk the layout mostly
  designs out. Not precluded as a future defense-in-depth addition.
- **Scrape CHANGELOG prose from milestone/epic commit messages.** Rejected:
  commit messages are terse engineering shorthand (`git log` / `aiwf
  history` audience), not release-notes prose; treating them as a CHANGELOG
  source would produce the wrong register.

## Consequences

- `wrap.md`'s template (`aiwfx-wrap-epic` skill) gains a `## Changelog
  entry` section, adjacent to `## Milestones delivered`.
- `aiwfx-wrap-epic` step 7's instructions change from "distill into a new
  CHANGELOG paragraph" to "copy `## Changelog entry` verbatim into
  `CHANGELOG.md`."
- G-0368 is redirected from an audit-first plan toward implementing this
  section-split and copy step.
