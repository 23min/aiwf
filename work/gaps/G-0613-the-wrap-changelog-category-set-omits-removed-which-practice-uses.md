---
id: G-0613
title: the wrap changelog category set omits Removed, which practice uses
status: addressed
addressed_by_commit:
    - edf2b2171
---
## What's missing

Two shipped surfaces name the category set a changelog entry is written under,
and both name three of Keep a Changelog's six: the `wrap.md` scaffold in
`aiwfx-wrap-epic`, and step 4 of `wf-patch`.

An epic or a patch that retires a verb, a flag, or a section has no listed
category to write under. The author either picks a wrong one, or invents a
heading the next reader cannot predict.

This repo's own `CHANGELOG.md` already exceeds the three. Released sections
carry `### Changed (breaking)` and `### Security`, one carries `### Internal`,
and the section awaiting release carries `### Removed` — written by the patch
that retired the milestone spec's `## Work log`.

## Why it matters

The cost is small per instance and permanent: a closed set on a shipped surface
is followed by consumers who have no way to know it was narrowed rather than
transcribed. Keep a Changelog itself defines six categories; shipping three
without saying why reads as the whole vocabulary.

## Direction

Settled in D-0088: the six, with a parenthetical after the category — `Changed
(breaking)`, `Changed (internal)` — read as a note on that category rather than
a seventh one. Work with no user-visible delta goes under `Changed`. Both
surfaces carry it.

D-0031 is not amended and stays `accepted`. It settled where the changelog prose
is authored, and named the headings once, parenthetically, as "the
Keep-a-Changelog heading shape already in use today" — a description of practice
at the time rather than a closed set, and its Reasoning and Consequences do not
mention categories at all. The three-heading list in the shipped surfaces is a
transcription of that illustration, which is the second of the two readings this
gap posed.

No check pins the set. D-0070 retires prose-content assertions over shipped
surfaces, so this is held at review like the rest of what those rituals
instruct.

## References

D-0088 settles the set. Found while reviewing the patch that closed G-0368,
which shipped the three-category list into the `wrap.md` scaffold.
