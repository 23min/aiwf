---
id: G-0541
title: The guidance's template path resolves for two of six kinds
status: addressed
priority: medium
addressed_by_commit:
    - 4d94db6c2
---
## What's missing

Two template systems carry the same name, and the shipped guidance points at the
one that mostly is not there.

`entity.BodyTemplate` returns a per-kind section scaffold for all six kinds, and
that is what `aiwf add` writes. Beside it sits a second, richer set: the
prose-guided `.claude/templates/*.md` the rituals materialize. Those cover four
kinds — `epic-spec.md`, `milestone-spec.md`, `adr.md`, `decision.md`. **Gap and
contract have none.**

The always-on guidance fragment, which materializes into every consumer repo,
instructs an assistant to create an entity "filling the body from
`.claude/templates/<kind>.md` (`aiwf update` if absent)". Substituting each kind
against what ships, the path resolves for **two of six**: `adr` and `decision`.
It misses `epic` and `milestone` on the `-spec` suffix, and `gap` and `contract`
because no such file exists.

The parenthetical compounds it. `aiwf update` will never produce the four missing
files, so the remedy offered reads as "your tree is stale" where the truth is
"this file was never going to exist" — and the reader most likely to follow it is
an assistant with no prior of what the directory holds.

## Why it matters

The failure is silent at every step. No check fires, `aiwf doctor` reports the
rituals materialized, and the consumer sees a healthy tree. An assistant that
follows the instruction finds nothing, and then either invents a structure or
skips the body — both of which land in the planning record and neither of which
anything catches.

Gap bodies show the result. Beyond the two sections `aiwf add` mandates, the
corpus carries more than a dozen distinct headings for overlapping purposes —
`## Problem` beside `## What's missing`, and `## Resolution shape` beside
`## Fix shape`, `## Proposed fix shape` and `## Where to fix`. Nothing anchors
them because for gaps there is nothing to anchor to.

It is worse for a consumer than for this repo. Here the conventions are visible
in a thousand neighbouring entities; a fresh tree has none, so the guidance is
the whole of what a consumer's assistant has to go on, and for four of six kinds
it points at a file that is not there.

## Resolution shape

Three parts, and the third is a decision rather than work.

1. **Write the two missing templates**, matching the depth of the four that
   exist rather than restating `BodyTemplate`'s bare headings.
2. **Make the guidance's path pattern resolve.** Either rename so `<kind>.md`
   holds for all six, or drop the pattern and let the guidance name the files.
   The rename is cleaner and breaks any consumer who references the current
   names; naming them is safe and leaves the reader to track six spellings.
3. **Decide whether the two systems should stay two.** A scaffold in the binary
   and a prose template on disk can legitimately differ in depth, but today they
   also differ in *coverage* and nothing says which is authoritative. Settling
   that is what stops the pair drifting apart again.

Not proposed: a check that every kind has a template. The absence here is a
missing artifact, not a missing detector, and a detector would pass the moment
the files land.
