---
id: G-0532
title: entity-id-narrow-width reads only the filename, not frontmatter
status: addressed
priority: medium
discovered_in: M-0290
addressed_by_commit:
    - 3aa8a06
---
## Problem

An entity carries its id twice: in the on-disk filename
(`work/gaps/G-0100-paint.md`) and in frontmatter (`id: G-0100`).

`entity-id-narrow-width` tested only the filename, so a narrow
frontmatter id under a canonical filename produced no finding of any
code — the entity loaded, `aiwf list` showed it, and `aiwf check`
exited 0. Measured: a gap file at `work/gaps/G-0200-reverse.md`
carrying `id: G-200` yielded zero findings.

No other rule closed it. `idPathConsistent` canonicalizes both sides
before comparing, so `G-200` and `G-0200` compare equal and it reports
no mismatch; `frontmatterShape` validates against the kind's grammar
floor, which admits narrow ids permanently. A width-only divergence
between the two axes was invisible by construction on every path.

## Resolution

`entity-id-narrow-width` reads both axes and judges each independently.
It is the only rule that tests an entity's own id width, so whatever it
declines to read goes unreported by everything. `idPathConsistent` keeps
its width-blindness unchanged — that tolerance exists so a tree carrying
both widths for one entity draws no spurious mismatch, and it stays
correct once width is owned elsewhere.

Three consequences settle the cases a two-axis rule would otherwise
leave ambiguous:

- **One finding per narrow entity, not per narrow axis.** One entity is
  one defect with one fix.
- **The message quotes only spellings that are actually narrow** — the
  filename id, the frontmatter id, or both when they are narrow and
  disagree. Printing a canonical id and calling it narrow would
  contradict itself at the one seam an operator reads to locate the
  file. When both axes are narrow at the same spelling, no axis is
  named, since naming one implies the other is clean.
- **An `id:` below the kind's grammar floor is malformed, not narrow.**
  That case routes to `frontmatterShape`, which names the expected
  format, rather than stacking a second finding on top. It mirrors the
  filename axis, where `entity.IDFromPath` rejects a sub-floor path
  before the width test sees it.

The two axes never gate each other. The loader admits filenames
`IDFromPath` rejects, and a rejected filename yields the empty id, which
is not narrow — so the path axis drops out on its own and a narrow `id:`
beneath it is still reported.

## Not this gap

The filename axis is correct and pinned, and its severity is unchanged:
error, per ADR-0008, for any narrow id in the active tree. Archive
entries stay excluded on both axes, permanently — no verb widens an id
in place, so a repo that archived entities before canonical width holds
narrow ids under `<kind>/archive/` forever, and the loader's narrow read
tolerance is what keeps live cross-references into them resolving.

## Provenance

Found 2026-08-03 by two independent reviewers during M-0290's wrap
review, which retired the width-migration verb. The condition predated
that milestone — the rule read the filename before it and after.
