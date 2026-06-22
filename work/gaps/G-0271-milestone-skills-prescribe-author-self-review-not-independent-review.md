---
id: G-0271
title: Milestone skills prescribe author self-review, not independent review
status: open
discovered_in: M-0171
---
## Problem

The milestone lifecycle skills make code review the *author's* job, not an
independent one. `aiwfx-start-milestone` step 7 ("Self-review before declaring
complete") has the implementing agent run the `wf-review-code` checklist against
its own diff, and `aiwfx-wrap-milestone` bundles the implementation without a
prescribed independent-review step. Nothing in the ritual dispatches the
`reviewer` agent (or any fresh perspective) over the held diff before the wrap
commit.

Author self-review carries the well-known author-blindness failure mode: the
agent that wrote the code is the worst-positioned reader of its own assumptions,
especially on multi-package changes and on the one or two non-obvious design
calls a milestone makes. This contradicts the kernel ethos that correctness must
not depend on the implementing agent getting it right unaided — the same reason
`aiwf check` and the pre-push hook exist. In practice the independent pass is
*frequently skipped* because the skill never asks for it; it surfaced here only
because the human explicitly asked whether independent review was in the plan.

## Direction (converge at the milestone)

Make an independent review a *prescribed* step in the milestone lifecycle —
candidate shapes, to decide later:

- `aiwfx-wrap-milestone` dispatches the `reviewer` agent (running
  `wf-review-code`) over the milestone diff before the wrap commit and surfaces
  its verdict as a gate input. Read-only, so no worktree bootstrap.
- `aiwfx-start-milestone` step 7 is reframed from "self-review" to "self-review
  + independent review," naming the subagent dispatch explicitly.
- An optional `wf-rethink` pass on any unit that embedded a non-obvious design
  decision during the milestone.

Whichever lands, the property is: a milestone does not reach wrap on the
implementing agent's self-assessment alone.

## Provenance

Surfaced during M-0171's pre-wrap planning (E-0043): the implementing agent's
plan listed only ritual step-7 self-review; the human caught that neither an
independent code review nor `wf-rethink` was in the plan, asked for both, and
asked to file this gap against the skills shortcoming.
