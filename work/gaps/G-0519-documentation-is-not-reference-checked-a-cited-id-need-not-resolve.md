---
id: G-0519
title: 'Documentation is not reference-checked: a cited id need not resolve'
status: open
priority: medium
---
## Problem

Nothing checks that a digit-bearing id written in documentation names a real
entity. The slug rule shipped alongside this gap closes the exact half — an id
written with a slug that contradicts the entity it names — but an id carrying
no slug is unchecked, whether it resolves to nothing at all or resolves to
something entirely unlike what the surrounding prose describes.

The motivating instance: a worked example in the workflows guide borrowed a
real ADR's id for a fictional decision about OAuth passkeys, four of its five
occurrences written bare. Width could not see it, because the id was already
canonical.

## The convention this rests on

Illustrative ids are written `<prefix>-NNNN`. Treat that as exhaustive and a
digit-bearing id in documentation is, by definition, a citation — at which
point requiring it to resolve is coherent rather than arbitrary.

## Why this is larger than it looks

Requiring resolution inherits every question `body-prose-id` had to settle,
and the answers are not obviously the same for documentation:

- **Cross-branch ids.** An id allocated on an unmerged branch resolves
  nowhere locally. The entity-body rule answers this with a non-blocking
  `cross-branch-pending` subcode; documentation would need the same, or a
  reason not to.
- **Archived narrow ids.** Read tolerance is permanent, so a doc citing an
  archived entity at its genuine narrow width is correct and must not fire.
- **Corpus breadth.** The doc corpus is configured and currently small.
  Resolution-checking a wide corpus surfaces far more, including citations
  that were accurate when written and were later archived or renamed.

## Residue this still will not catch

A bare canonical-width id used as fiction, where that id happens to name a
real entity, and no slug is written. Distinguishing it from a citation would
mean comparing surrounding prose against the entity's title, and any
proximity heuristic fires on legitimate text that mentions an id and a quoted
string in one sentence. The placeholder convention plus review covers this;
the gap does not propose to automate it.
