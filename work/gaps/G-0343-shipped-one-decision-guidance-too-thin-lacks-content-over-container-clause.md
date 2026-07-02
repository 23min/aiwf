---
id: G-0343
title: Shipped one-decision guidance too thin, lacks content-over-container clause
status: open
---
## Problem

Two related weaknesses in how the shipped guidance carries the
one-decision-at-a-time rule:

1. **The shipped bullet is thinner than the rule.**
   `internal/skills/embedded-guidance/aiwf-guidance.md:31` (materialized into every
   consumer's `.claude/aiwf-guidance.md`) says *"present them one at a time with
   context and a recommendation."* The repo-dev statement it mirrors (`CLAUDE.md:23`)
   is richer: *"context, pros/cons, risks, your plain lean, then a numbered option
   list."* The shipped version drops **pros/cons, risks, and the lean-with-argument**
   — exactly the reasoning a human needs in order to decide. A consumer whose only
   decision-guidance is the materialized fragment gets the weak version.

2. **No content-over-container clause.** Neither the shipped guidance nor
   `CLAUDE.md` says the decision *content* (context, pros/cons, risks, and the
   argument behind the lean) is the rule while the *container* (prose vs an
   `AskUserQuestion` card) merely serves it. The harness's `AskUserQuestion` tool
   nudges the opposite way — up to four questions batched per call, a short per-option
   `description`, and the lean reduced to a `(Recommended)` tag (the conclusion
   without the argument). An AI following the tool's affordance can present a terse,
   batched card and rationalize it as compliant, defeating the rule. The one existing
   `AskUserQuestion` mention (`CLAUDE.md:25`) is scoped to *gate* bundling, not
   decision presentation, so it does not cover this.

Observed downstream: humans getting card-shaped forks with too little context — no
pros/cons, no lean, no argument.

## Direction (for the milestone that addresses this)

- Enrich the shipped bullet (`aiwf-guidance.md:31`) to carry the full content
  requirement: context, pros/cons, risks, a plain lean **with its argument**, and a
  numbered option list including an escape option. Keep the
  `PolicyM0211GuidanceOperatingAnchors` `one-decision-at-a-time` anchor fragment
  (`one thing at a time`) intact through the edit.
- Add a **content-over-container** clause (shipped guidance + `CLAUDE.md:23`): the
  reasoning is the deliverable; the container serves it. Prose is the default for
  reasoning-heavy forks; reserve `AskUserQuestion` cards for a visual/structural
  comparison of concrete artifacts (the tool's `preview` feature) or a genuinely
  light either/or — and even then carry the recommendation. Never let a card's
  terseness drop the pros/cons/lean/argument, and never batch questions.
- Land a structural test under `internal/policies/` (or extend
  `PolicyM0211GuidanceOperatingAnchors` with the enriched fragments) as the AC's
  mechanical evidence.
- Re-materialize via `aiwf update`.

## Not in scope

- The operator's *format preference* (prose over cards) — a personal collaboration
  call that belongs in the operator's own base guidance layer, not shipped by aiwf.
  Same audience split as G-0341: aiwf ships the *content-protection substance*, not
  the *format opinion*.
- `CLAUDE.md:25` (the gate-bundling prohibition) — correct as-is; a different
  concern.

Sibling of G-0341 / G-0342 — same class: a shipped artifact carrying less than, or
diverging from, the kernel's actual collaboration rule.
