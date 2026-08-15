---
id: G-0591
title: Live prose defers to work that was declined, and nothing catches it
status: open
priority: medium
---
## What's missing

Live prose defers work to entities that were later declined, and nothing reports
it. A sentence saying a concern is *deferred to* or *tracked as* some entity
makes a promise about the future; when that entity is later cancelled, rejected,
superseded or closed `wontfix`, the promise has no destination and the sentence
still reads as though it does.

Measured instances, each verified against the target's current status:

- `docs/design/performance.md:230` — "Deferred levers carried forward (profiled,
  unstarted)" names G-0323 and G-0325. Both are `wontfix`. The same document
  records a sibling lever's decline correctly a few lines earlier, so it is
  internally inconsistent about which levers are alive.
- `docs/design/performance.md:228` — "Reopening for that shape is justified
  (deferred to G-0340)". G-0340 is `wontfix`.
- `CLAUDE.md:51` — the documentation tiering "does not drift-check against
  `docs/`'s actual contents at runtime (tracked as a kernel-rule follow-on in
  G-0092)". G-0092 is `addressed` and archived, and the follow-on it names was
  marked out of scope inside it. This line is in the always-loaded file.

The distinction that matters is between a deferral to work that was **delivered**
and one to work that was **declined**. A pointer at a `done` milestone is merely
dated — the work happened, the sentence has aged. A pointer at declined work is a
hole: the concern was handed to something that then refused it, and no surface
says so.

A first scan of live prose — the normative documents, the active ADRs and
decisions, the open gaps, and `CLAUDE.md` — found 24 forward-tense citations
naming 17 terminal targets, of which the declined subset is the defect class. That
count is a floor rather than a measurement: the phrasing at `performance.md:230`
does not match the pattern that found the others, which is itself evidence that
enumerating the shapes is part of the work.

## Why it matters

This is the class that survives every existing check. `refs-resolve` walks
frontmatter reference fields and never reads prose. The `doc-id-*` rules scan only
what `docs.paths` names, which here is two files. `wf-doc-lint` finds broken
links, and every sentence above parses, links fine, and is false.

It is also the class that makes extraction unsafe. A finding moved out of a
terminal entity into a durable document is only worth moving if the document
stays true; measured here, one such pointer went stale twelve days after the
decline it refers to, and another has been wrong for months in the file every
session loads. Distilling knowledge into a steering surface that nothing
re-derives converts a fact nobody reads into a fact everybody reads and nobody
checks.

The shape that fits is the one the repository already uses for comment staleness:
a whole-tree scan with a named escape, reporting rather than blocking, so the
number can be driven to zero and held there. Severity is a real question — a
deferral to delivered work is not a defect, and reporting all 17 would teach
readers to ignore the finding.
