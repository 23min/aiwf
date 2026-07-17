---
id: G-0368
title: wrap.md gains a Changelog entry section, copied verbatim at epic wrap
status: open
priority: medium
---
## Problem

`aiwfx-wrap-epic` step 7 authors the epic's `CHANGELOG.md` `[Unreleased]` entry
independently from `wrap.md`'s own `## Summary` section, even though both are
written in the same wrap sitting and describe the same epic. Nothing links the
two, so the CHANGELOG entry can silently drop a milestone's user-visible delta
without the `wrap.md` summary catching it. `aiwfx-wrap-milestone` has no
CHANGELOG step of its own (only epic wrap ever touches `CHANGELOG.md`), and
`aiwfx-release` only moves whatever already sits under `[Unreleased]` — it
never synthesizes entries from milestone or wrap history.

## Direction

Per D-0031: `wrap.md` becomes the single point of authorship for what to tell
people about an epic, split by audience into two adjacent sections —
`## Summary` (internal, unchanged) and a new `## Changelog entry` (written for
a release-notes reader, Keep-a-Changelog heading shape, optional bullet per
milestone), sitting directly beneath `## Milestones delivered`.
`aiwfx-wrap-epic` step 7 changes from freely re-authoring a CHANGELOG
paragraph to copying `## Changelog entry` verbatim into `CHANGELOG.md` under
`[Unreleased]`. `aiwfx-wrap-milestone` stays changelog-free.

## Scope

- Add `## Changelog entry` to the `wrap.md` template in `aiwfx-wrap-epic`'s
  `SKILL.md` (step 1's scaffold), directly beneath `## Milestones delivered`.
- Rewrite step 7's instructions: copy `## Changelog entry` verbatim into
  `CHANGELOG.md` instead of distilling a new paragraph.
- Check the "Out of scope" note still holds the copy-not-synthesize boundary.
- A hand-written pinning test under `internal/policies/`, per this repo's
  `skill-edit-structural-test-backstop` convention for `SKILL.md` edits.

## Provenance

Direction settled in D-0031 (2026-07-06).
