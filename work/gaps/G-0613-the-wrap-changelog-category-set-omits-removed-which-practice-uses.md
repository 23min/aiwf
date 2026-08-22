---
id: G-0613
title: the wrap changelog category set omits Removed, which practice uses
status: open
---
## What's missing

D-0031 fixed the wrap changelog entry's heading shape at three Keep-a-Changelog
categories — Added, Changed, Fixed — and the `wrap.md` template now ships that
closed set. This repo's own `CHANGELOG.md` already exceeds it: released sections
carry `### Changed (breaking)`, a category the set does not name.

An epic that retires a verb or a flag has no listed category to write under. The
author either picks a wrong one, or invents a heading the next reader cannot
predict.

## Why it matters

The cost is small per instance and permanent: a closed set on a shipped surface
is followed by consumers who have no way to know it was narrowed by a decision
rather than by the upstream convention. Keep a Changelog itself defines six
categories; shipping three without saying why reads as the whole vocabulary.

## Direction

This is a D-0031 amendment, not a review correction — the decision named three
categories deliberately and is `accepted`, so widening the set means revisiting
it rather than editing the template underneath it.

The question to settle: is the three-category set a real constraint (an epic's
delta should be summarisable under one of three) or an incomplete transcription
of Keep a Changelog? If the former, the template should say so, and `Removed`
work should be expressed as `Changed`. If the latter, widen the set, and decide
separately whether `Changed (breaking)` is a category of its own or a convention
within `Changed`.

The wrap ritual is now the only surface naming a category set, so whichever way
this settles, it settles in one place.

## References

D-0031 fixed the three-category set. Found while reviewing the patch that closed
G-0368, which shipped that set into the `wrap.md` template.

The gap originally cited a second surface — `aiwfx-release`'s release-section
template, which offered four categories including `Removed`. The patch closing
G-0611 and G-0612 deleted that template, so the release ritual now names no
category set at all. That removed the two-rituals-disagree framing without
touching the question above, which stands on this repo's own `Changed (breaking)`
usage.
