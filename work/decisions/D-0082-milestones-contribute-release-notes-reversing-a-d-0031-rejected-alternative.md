---
id: D-0082
title: Milestones contribute release notes, reversing a D-0031 rejected alternative
status: proposed
---
> **Date:** 2026-08-31 · **Decided by:** human/peter

## Question

D-0031 considered having each milestone contribute a note for the epic wrap to
fold into its changelog entry, and rejected it. M-0326 built that. Does the
rejection still hold, and if not, what in D-0031 changes?

## Decision

The rejection does not hold. Each milestone now records its own user-visible
delta at its wrap, and the epic wrap composes its changelog entry from those
notes instead of reconstructing one from milestone titles and merge SHAs.

D-0031's core holding is unchanged and stays accepted: the changelog has a
single producer, the epic wrap, and `## Changelog entry` is authored once and
copied verbatim. What changes is where that producer is told to get its
material. A milestone still writes nothing to `CHANGELOG.md`.

## Reasoning

D-0031 rejected the alternative on the grounds that it bought nothing "the
wrap.md funnel doesn't already give for free in the same sitting". That premise
was reasonable and is now measured false. Cutting v0.34.0 shipped three changes
with nothing written about them, two of them on `docs(` commits — a prefix that
means "nothing user-visible" in most repos and the opposite here, since guidance
and rituals ship as product. The funnel did not give it for free; it gave an
author at the end of an epic reconstructing from titles, which is where the
three went missing.

The second half of the rejection — that it "makes milestone wrap a second
CHANGELOG-bound producer, breaking the single-producer pattern" — is avoided
rather than accepted. The milestone wrap writes a section in its own spec. Only
the epic wrap writes to `CHANGELOG.md`, exactly as D-0031 requires.

What this does not buy: nothing enforces that the epic wrap actually reads the
notes. The chain from note to changelog is ritual instruction with one
non-blocking warning at its first link, `milestone-done-empty-release-note`,
which reports a milestone reaching `done` with an unwritten note. The guarantee
that a shipped change cannot reach a release undescribed is not established here
and is tracked in G-0529.

## Consequences

- D-0031 stays `accepted`. Its rejected-alternatives list carries one entry that
  has since been revisited; this record is the forward pointer.
- `aiwfx-wrap-milestone` gains a `## Release note` step and is no longer
  changelog-free in the sense D-0031 used, though it remains changelog-free in
  the sense D-0031 required — it writes no changelog file.
- `aiwfx-wrap-epic` step 1 composes from the notes rather than from titles.
