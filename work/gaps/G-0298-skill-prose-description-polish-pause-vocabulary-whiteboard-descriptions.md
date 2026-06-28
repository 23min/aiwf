---
id: G-0298
title: 'Skill prose & description polish: pause vocabulary, whiteboard, descriptions'
status: open
---
## Problem

Low-severity prose and description defects across skills:

- **Prohibited "pause" vocabulary at completion boundaries.** CLAUDE.md and
  `.claude/aiwf-guidance.md` say never suggest the user pause. `aiwfx-plan-epic`
  says *"merge to main **and pause** if stopping here"*; `aiwfx-wrap-epic` ends
  with *"or **stop here**."* The forks themselves are legitimate (they sit at
  genuine completion boundaries where the work is done and the next step is the
  user's call), but the literal vocabulary collides with a standing rule the
  materialized skills are read against every session.
- **`aiwfx-whiteboard` description contradicts its body.** The frontmatter
  `description:` ends *"Read-only; no commit; no persisted artefact,"* but the
  body instructs writing a gitignored `WHITEBOARD.md` cache, and an anti-pattern
  explicitly blesses that cache. An LLM trusting the description won't write the
  cache the body requires. A stale cross-reference ("see Output cache below")
  also points the wrong direction (the section is above).
- **`wf-codebase-health` description is a near-twin of the global `code-health`
  skill** (an ai-dotfiles-provided skill present on this machine). The shared
  opening sentence makes skill selection a coin-flip whenever both are active.
- **`aiwf-retitle` description lists "rename the title,"** colliding with
  `aiwf-rename`'s primary trigger ("rename"). The body disambiguates, but the
  description is the selection surface.

## Decision

- `aiwfx-plan-epic` / `aiwfx-wrap-epic`: keep the completion-boundary forks but
  scrub the prohibited vocabulary — reframe "pause" / "stop here" as completion
  ("merge to main if planning is complete for now"; "for whatever's next").
- `aiwfx-whiteboard`: fix the description to state it writes a gitignored
  `WHITEBOARD.md` cache; fix the "below" cross-reference to "above". (The stale
  real-id tier examples are handled by the skill-body id-hygiene gap.)
- `wf-codebase-health`: differentiate the description — lead with its aiwf-ritual
  identity (the per-repo companion to `wf-review-code`'s per-diff gate) rather
  than the generic sentence shared with `code-health`. (The reconciliation of the
  global `code-health` skill in ai-dotfiles is a separate, out-of-repo decision.)
- `aiwf-retitle`: soften the "rename the title" trigger to "change/correct the
  title" to avoid the `aiwf-rename` collision.

## Scope

`aiwfx-plan-epic`, `aiwfx-wrap-epic`, `aiwfx-whiteboard`, `wf-codebase-health`,
`aiwf-retitle`.
