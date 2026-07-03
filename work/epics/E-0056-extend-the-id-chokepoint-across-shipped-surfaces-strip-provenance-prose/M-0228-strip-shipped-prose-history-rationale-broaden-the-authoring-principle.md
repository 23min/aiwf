---
id: M-0228
title: Strip shipped-prose history/rationale; broaden the authoring principle
status: draft
parent: E-0056
depends_on:
    - M-0227
tdd: advisory
---
## Goal

Shipped consumer surfaces carry no development history, provenance narrative, or
rationale/argumentation — only imperative, consumer-scoped instruction. This
repo's `CLAUDE.md` states the broadened authoring principle, so a future edit
that reintroduces history or rationale is held to it at review.

## Approach

Rewrite the history/rationale prose the extended check cannot catch:

- the statusline's provenance comments (the "superseding the earlier ..."
  narrative and the bare gap-id tags in `#` comments);
- the "the v1 separate tracking doc is gone" asides in `aiwfx-start-milestone`,
  `aiwfx-wrap-milestone`, and `templates/milestone-spec.md`;
- the "Why date and decided_by are in the body ..." argumentation blocks in
  `templates/adr.md` and `templates/decision.md` — reduce each to the imperative
  instruction, drop the "why".

Extend `CLAUDE.md` § "Skills policy" (the existing "Shipped skill bodies cite no
real entity id" paragraph) to name the full surface list and add the content
class: no development history, no provenance tags, no rationale or war-stories.
(The reference/dead-link discipline is owned separately by the doc-link
milestone, which encodes it in the `aiwfx-record-decision` skill.)

Where mechanizable, add a structural assertion (e.g. no `(G-NNNN)`-style
provenance tag in shipped prose) as the AC's evidence; the non-mechanizable
remainder rides the review backstop. Depends on the broadened check landing
first.

## Acceptance criteria

Sketch — formalized at start-milestone:

1. The named surfaces carry no development-history aside or rationale block — a
   structural assertion over the cleaned surfaces holds.
2. `CLAUDE.md` § "Skills policy" states the broadened authoring principle (full
   surface list plus the history / provenance / rationale content class), pinned
   by a structural test scoped to the named section.
