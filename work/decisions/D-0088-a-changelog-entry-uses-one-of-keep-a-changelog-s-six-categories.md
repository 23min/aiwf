---
id: D-0088
title: A changelog entry uses one of Keep a Changelog's six categories
status: proposed
relates_to:
    - G-0613
    - D-0031
---
> **Date:** 2026-09-11 · **Decided by:** human/peter

## Question

Which headings may a changelog entry be written under?

The rituals that write `CHANGELOG.md` name three — `Added`, `Changed`, `Fixed`.
Keep a Changelog, the convention they cite, defines six. An epic or a patch that
retires something has no listed heading to write under, so its author either
files it under a heading that misdescribes it or invents one the next reader
cannot predict.

The narrower set is not a decision anyone took. D-0031 settled where the prose
is authored, and named the headings once, parenthetically, as "the
Keep-a-Changelog heading shape already in use today" — a description of practice
at the time, not a closed set. Its Reasoning and Consequences do not mention
categories. The three-heading list in the shipped surfaces is a transcription of
that illustration.

Practice has outrun it. This repo's own changelog carries `Removed`, `Security`,
`Internal` and `Changed (breaking)` alongside the three, and the `Removed`
heading is in the section awaiting release.

## Decision

A changelog entry is written under one of Keep a Changelog's six categories:
`Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`.

A parenthetical after the category — `Changed (breaking)`, `Changed (internal)` —
is a note on that category, not a seventh one. It is available where it tells a
release-notes reader something the category alone does not, and nothing requires
it.

Work with no user-visible delta is written under `Changed`, qualified if useful.
`Internal` is not a category.

D-0031 is unaffected and stays `accepted`. Where the prose is authored, and that
it is copied rather than re-written, is its subject and does not change here.

## Reasoning

The two readings the question poses are not evenly supported. For "three is a
real constraint" the only evidence is the list itself; nothing states the
constraint or argues for it, and the repo that ships the list breaks it. For
"incomplete transcription" there is D-0031's own wording and the fact that the
categories appear nowhere in its argument.

Adopting the upstream six rather than inventing a set costs a reader nothing to
learn and leaves no gap where a category is needed but absent. Deprecated and
Security have no instance in this repo yet; they ship because a set that omits
whatever has not happened yet is the defect being fixed, and because a consumer
repo is not this one.

The qualifier rule is what keeps the six honest. Four uses of `Changed
(breaking)` are already in the tree, and reading them as violations of a
six-category set would be a rule written against practice for the second time.
Reading them as a note on `Changed` costs one sentence and describes what the
author meant.

Alternatives considered:

- **State that three is deliberate, and write removals as `Changed`.** Rejected:
  it needs an argument for the constraint, and there is none to make. It would
  also read to a consumer as aiwf narrowing a convention it cites by name, with
  no reason given.
- **Adopt the six and treat `Changed (breaking)` as a seventh category.**
  Rejected: a category set that grows by one entry each time someone needs a
  distinction is not a set. The parenthetical already reads as a modifier.
- **Keep `Internal` as a category.** Rejected: it names the absence of a
  user-visible delta, which is the one thing a release-notes reader does not
  need a heading for. The wrap ritual already says what to do with a wholly
  internal delta.

## Consequences

- The two shipped surfaces naming the set — the `wrap.md` scaffold in
  `aiwfx-wrap-epic`, and step 4 of `wf-patch` — carry the six and the qualifier
  rule.
- No check pins the set. A prose-content assertion over a shipped surface is
  retired by D-0070, so this is held at review like the rest of what those
  rituals instruct.
- Entries already written under `Internal` stay as written. Nothing rewrites a
  released section.
