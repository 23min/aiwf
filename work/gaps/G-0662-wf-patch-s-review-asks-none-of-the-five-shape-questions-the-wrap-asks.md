---
id: G-0662
title: wf-patch's review asks none of the five shape questions the wrap asks
status: open
discovered_in: M-0327
---
## What's missing

`aiwfx-wrap-milestone` step 2 asks five questions about the *shape* of a change —
Recurring obligation, Deletions, Same-outcome clusters, Compression, Over-guarding.
Every other check at wrap asks whether something is missing; these are the only ones
that ask whether something is surplus.

`wf-patch` asks none of them. Measured against
`internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`, each
of the five scores zero.

## Why it matters

A patch is the higher-traffic surface. Work that never becomes a milestone never
meets the only review lens that can shrink it, so the surplus the wrap ritual exists
to catch accumulates on the path that skips the wrap.

Porting them is blocked on G-0660: the compression lens's gate run cannot go red for
the deletions it proposes, so the lens currently reports a result its own oracle
cannot falsify. Copying it into `wf-patch` before that is repaired puts the defect on
the busier surface.
