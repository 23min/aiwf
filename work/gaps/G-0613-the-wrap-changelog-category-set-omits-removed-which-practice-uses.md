---
id: G-0613
title: the wrap changelog category set omits Removed, which practice uses
status: open
---
## What's missing

D-0031 fixed the wrap changelog entry's heading shape at three Keep-a-Changelog
categories — Added, Changed, Fixed — and the `wrap.md` template now ships that
closed set. Two surfaces already exceed it:

- `aiwfx-release`'s own release-section template offers four, including `Removed`.
- This repo's `CHANGELOG.md` carries `### Changed (breaking)` in released
  sections, a category neither set names.

An epic that retires a verb or a flag has no listed category to write under. The
author either picks a wrong one, or invents a heading the next reader cannot
predict.

## Why it matters

The cost is small per instance and permanent: a closed set on a shipped surface
is followed by consumers who have no way to know it was narrowed by a decision
rather than by the upstream convention. Keep a Changelog itself defines six
categories; shipping three without saying why reads as the whole vocabulary.

It also puts two shipped rituals in disagreement about the same file, which is
the shape that produced the other two gaps filed alongside this one.

## Direction

This is a D-0031 amendment, not a review correction — the decision named three
categories deliberately and is `accepted`, so widening the set means revisiting
it rather than editing the template underneath it.

The question to settle: is the three-category set a real constraint (an epic's
delta should be summarisable under one of three) or an incomplete transcription
of Keep a Changelog? If the former, the template should say so, and `Removed`
work should be expressed as `Changed`. If the latter, align the wrap set with
`aiwfx-release`'s four and decide separately whether `Changed (breaking)` is a
category or a convention within `Changed`.

Either way the two rituals should end up naming the same set.

## References

D-0031 fixed the three-category set. Found while reviewing the patch that closed
G-0368, which shipped that set into the `wrap.md` template.
