---
id: G-0532
title: entity-id-narrow-width reads only the filename, not frontmatter
status: open
priority: medium
discovered_in: M-0290
---
## Problem

An entity carries its id twice: in the on-disk filename
(`work/gaps/G-0100-paint.md`) and in frontmatter (`id: G-0100`).

`entity-id-narrow-width` reads only the filename. A narrow frontmatter
id under a canonical filename produces no finding of any code — the
entity loads, `aiwf list` shows it, and `aiwf check` exits 0.

The rule that compares the two sides does not close the gap either.
`idPathConsistent` canonicalizes both before comparing, so `G-200` and
`G-0200` compare equal and it reports no mismatch. The width-only
divergence is invisible by construction on both paths.

Measured: a gap file at `work/gaps/G-0200-reverse.md` carrying
`id: G-200` yields zero findings.

## Why it is not simply a bug to fix

Reading both axes forces a decision the current rules avoid.

When the two spellings disagree in width, it is not obvious whether that
is one finding or two, nor which spelling the message should quote. The
filename axis already answers this — it quotes the path-derived id,
because the path is what a reader greps for.

More importantly, `idPathConsistent`'s width-blindness is deliberate: it
exists so that a tree carrying both widths for the same entity does not
draw a spurious mismatch. Widening the width rule without revisiting
that decision produces either double-reporting on one file or two rules
disagreeing about whether the same file is well-formed.

## Direction

Two coherent resolutions, and the choice is the work:

- **Read both axes in `entity-id-narrow-width`**, reporting the narrow
  spelling and leaving `idPathConsistent` alone. Requires deciding the
  disagreement case explicitly.
- **Declare frontmatter width out of scope for this rule** and give the
  frontmatter axis to whichever rule owns frontmatter shape, stating in
  both rules' documentation that the split is intentional.

Either way the outcome is a fixture whose `id:` is narrower than its
path, asserting whichever behavior is chosen.

## Not this gap

The filename axis is correct and pinned. So is the message it quotes:
the rule tests the filename's width, so quoting frontmatter would print
a canonical id and call it narrow.

## Provenance

Found 2026-08-03 by two independent reviewers during M-0290's wrap
review, which retired the width-migration verb. The condition predates
that milestone — the rule read the filename before it and after.
