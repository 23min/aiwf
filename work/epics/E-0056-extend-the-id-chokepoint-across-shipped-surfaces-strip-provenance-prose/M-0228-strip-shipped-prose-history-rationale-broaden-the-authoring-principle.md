---
id: M-0228
title: Strip shipped-prose history/rationale; broaden the authoring principle
status: in_progress
parent: E-0056
depends_on:
    - M-0227
tdd: advisory
acs:
    - id: AC-1
      title: CLAUDE.md Skills-policy states the broadened authoring principle
      status: open
---
## Goal

Shipped consumer surfaces carry no development history, provenance narrative, or
rationale/argumentation — only imperative, consumer-scoped instruction. This
repo's `CLAUDE.md` states the broadened authoring principle, so a future edit
that reintroduces history or rationale is held to it at review.

## Approach

Rewrite the history/rationale prose the extended check cannot catch:

- the statusline's provenance comments (the "superseding the earlier ..."
  narrative and any remaining prose provenance in `#` comments);
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

Depends on the broadened check (`M-0227`) landing first: it mechanically removes
every *id-bearing* provenance tag from the scanned surfaces, so what this
milestone rewrites is the residual *non-id* history/rationale prose —
"superseding the earlier ...", "the v1 ... is gone", the "why" blocks — which
carries no stable machine-detectable shape.

## Acceptance criteria — evidence split (load-bearing; do not blur)

**Only mechanizable claims become met-with-a-test ACs. The prose-quality cleanup
rides the `aiwfx-wrap-milestone` review backstop and is deliberately NOT an AC.**

Why the split is load-bearing, not a shortcut: the repo rule "AC promotion
requires mechanical evidence" binds even at `tdd: advisory` — "I read it and it
looks right" is not evidence, and a doc-shaped AC needs a structural assertion
scoped to a named section. But "reads as imperative, consumer-scoped, no
war-stories" is a judgment no structural assertion can pin. Once `M-0227` has
stripped the id-bearing tags, the remaining history/rationale prose has no
machine-detectable shape. Forcing it into an AC would demand either a vacuous
test or a rule violation. So it is delivered as the milestone's work and verified
at the wrap review — never promoted to `met` against a test.

Sketch — formalized at start-milestone:

1. **(met-with-a-test AC)** `CLAUDE.md` § "Skills policy" states the broadened
   authoring principle — the full surface list plus the history / provenance /
   rationale content class — pinned by a structural assertion scoped to that
   named section (not a bare substring grep).

2. **(review backstop — NOT an AC)** The named surfaces read as imperative,
   consumer-scoped instruction with no development-history aside, rationale
   block, or war-story. Delivered work, verified at wrap review, not against a
   test.

A second met-with-a-test AC may be added at start-milestone *only* if a
non-brittle structural assertion presents itself (e.g. a targeted absence guard
over a specific cleaned file). If none does, this milestone ships with the single
mechanizable AC above — and that is correct, not a coverage gap.

### AC-1 — CLAUDE.md Skills-policy states the broadened authoring principle

