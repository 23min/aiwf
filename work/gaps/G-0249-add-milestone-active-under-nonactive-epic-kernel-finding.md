---
id: G-0249
title: add milestone-active-under-nonactive-epic kernel finding
status: open
---
## What's missing

The kernel has the `epic-active-no-drafted-milestones` finding but no
reciprocal guard for the inverse: a milestone promoted to `in_progress`
under a parent epic that is still `proposed` (or otherwise non-active).
A CLI-direct `aiwf promote M-NNNN in_progress` that bypasses the
`aiwfx-start-milestone` ritual flow lands with no finding, even though
the milestone is now in-flight under an epic that was never activated.

## Why it matters

The ritual skills route the operator through `aiwfx-start-epic` before
`aiwfx-start-milestone`, but per the kernel principle "framework
correctness must not depend on the LLM's behavior" a guarantee that
only holds when the skill flow is followed is not a guarantee. A
reciprocal `aiwf check` finding — e.g. `milestone-active-under-nonactive-epic`,
at warning or refusal severity — would catch the CLI-direct path that
skips the skill. The signal is already available: the kernel tracks
`ParentEpicStatus`, so the rule can read the parent epic's status
directly without new plumbing. This is the optional hardening flagged
alongside the `aiwfx-plan-milestones` Next-step doc bug (G-0248); it is
filed separately because it is a distinct subsystem — a new check-engine
rule plus tests — rather than a skill-content edit. Lower priority than
G-0248: the doc fix removes the mis-routing for skill-driven flows,
while this guard backstops the CLI-direct path the skills cannot reach.
