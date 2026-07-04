---
id: G-0349
title: wf-patch defaults to in-place branch, not the in-repo-worktree placement
status: wontfix
---
## What's missing

The `wf-patch` ritual's shipped skill (step 2, "Create a descriptive branch from
the project's mainline") cuts an in-place branch in the main checkout. It never
adopts the in-repo worktree-placement convention accepted in ADR-0023, which
`aiwfx-start-epic` follows: worktree placement defaults to in-repo under the
resolved `worktree.dir`, with main-checkout and sibling as overrides.
`aiwfx-start-milestone` inherits the epic's worktree, so both epics and
milestones run implementation on a separate worktree by default — only
`wf-patch` does not. Its only worktree mentions are passive wrap-cleanup
("remove the worktree if one was used"), never a placement default.

## Why it matters

The inconsistent default is a live hazard, not cosmetic. An in-place patch branch
shares the main checkout, so a second agent (or a scheduled task) running `aiwf`
verbs in the same working directory commits onto the patch branch — observed
directly, when a concurrent session's `aiwf add gap` landed on top of an
in-flight `wf-patch` fix, forcing a carry-both-to-mainline reconciliation. Epics
and milestones avoid this by isolating implementation in a worktree; `wf-patch`
should default the same way. Fix: make `wf-patch` step 2 cut the branch in an
in-repo worktree under the resolved `worktree.dir` by default, mirroring
`aiwfx-start-epic`'s placement step and its main-checkout / sibling overrides,
and remove the worktree at wrap. Shipped-doc constraint: the skill body must not
leak aiwf-internal ids or paths into consumer-facing prose (the `skill-body-id`
rule) — reference the placement convention through the sanctioned doc-link
carve-out (descriptive visible text, the id riding in the link destination),
exactly as `aiwfx-start-epic` does. A companion concern tracks internal-id
leakage across shipped consumer surfaces more broadly.
