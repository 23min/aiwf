---
id: G-0634
title: aiwf worktree add has no documented inverse, and wrap-epic inherits the silence
status: open
---
## What's missing

`aiwf worktree` offers `add` and no other subcommand, and nothing in the tree
says what undoes it. The `aiwf-worktree` skill does not mention removal.
Neither does ADR-0023, which chose the in-repo placement, nor any design doc.
The repo's own verb-design rule requires an answer — another invocation of the
same verb, a terminal transition, or a written "you can't, deliberately, and
here is why" — and none is recorded.

The consequence is that disposal was reinvented per ritual. `wf-patch` and
`aiwfx-wrap-milestone` each carry their own `awk` block resolving a worktree
path from a branch name, followed by `git worktree remove`, followed by the
branch delete, each re-explaining the same two ordering traps: git refuses to
delete a branch a worktree still holds, and you cannot remove the worktree you
are standing in. `aiwfx-wrap-epic` carries no such step at all. It mentions
worktrees only to resolve the *merge target's*, so the epic's own worktree is
created by default — `aiwfx-start-epic` places one in-repo unless told otherwise
— and never disposed of.

Measured 2026-08-25 after E-0088 closed: its worktree was still on disk holding
a merged branch, at 160 MB.

The likely cause is a coupling. Both rituals that clean up treat *remove the
worktree* and *delete the branch* as one act — same paragraph, same ordering
caveat. `aiwfx-wrap-epic` decided deliberately not to delete the branch, and
says so with its reason: local branches stay so history keeps its labels. Read
as a pair, dropping the branch delete drops the worktree removal with it. The
two are separable — `git worktree remove` touches no refs — but no ritual says
so, so nothing marks the second half as a separate decision that still needs
making.

## Why it matters

Locally the cost is small and recurring: an epic's worktree outlives the epic,
and nothing tells the operator it is theirs to remove. The branch policy at
least states its rationale, so an operator pruning branches knows the ritual
left them deliberately. The worktree gets silence, which reads as an oversight
whether or not it was one.

The durable reason is the missing inverse itself. A verb whose disposal is
undocumented gets its disposal re-derived by every caller, and re-derivation is
why one of the three callers has none. Adding a fourth hand-rolled copy to
`aiwfx-wrap-epic` would fix the symptom and leave that intact.

## Resolution shape

**Document the boundary; do not build `aiwf worktree remove`.**

The inverse the verb rule accepts is already available and only needs writing
down. `aiwf worktree add` does two things: `git worktree add`, and materializing
aiwf's gitignored rituals into the result. The second needs no undo, because
removing the directory removes the artifacts with it. The first is ordinary git
lifecycle. So the answer is *`git worktree remove`, deliberately unwrapped* —
aiwf's contribution to creation was materialization, not lifecycle.

Building the verb would bet against where every host is going. Measured
2026-08-25 from vendor documentation: Codex manages worktree lifecycle itself,
creating one from the branch selected in a chat, keeping the most recent fifteen
Codex-managed worktrees, and deleting them automatically when chats are archived
or space is needed. GitHub's Copilot app gives every agent session its own
worktree and creates and removes it per session. Claude Code offers worktree
isolation as a tool argument, which this repo denies at a `PreToolUse` hook after
it was observed to drop work into the live tree instead (G-0099).

Two of those reap on schedules aiwf cannot see. An aiwf-created worktree can be
deleted underneath it, and an aiwf disposal verb can delete one the host believes
it owns. That is contention in both directions, silent in both directions.

It is also ruled out by aiwf's own stated posture, in
[`agent-agnostic-execution-topology.md`](../../docs/initiatives/agent-agnostic-execution-topology.md):
*"aiwf informs and validates execution topology; it does not prescribe or own the
user's workflow"*, and explicitly, *"The initiative is therefore not 'make aiwf
manage worktrees.'"*

What the written boundary should carry, beyond naming `git worktree remove`:

- **The portability problem is materialization, not lifecycle.** aiwf's skills
  are gitignored, so a worktree created by a host rather than by `aiwf worktree
  add` contains no rituals. Codex's mechanism for this is a repo-root
  `.worktreeinclude` listing ignored paths to copy through; aiwf's is
  `aiwf update` plus `aiwf doctor --root <path>` to confirm. Which one a
  consumer should reach for is the seam worth specifying, and it is the only
  part of this that is aiwf's to answer.
- **`worktree.dir` defaults to a path named for one host.** The default sits
  under `.claude/`, which is correct for the host aiwf targets today and wrong
  for any other. The adapter question that owns this is already captured in
  [`agent-host-artifact-adapters.md`](../../docs/initiatives/agent-host-artifact-adapters.md),
  whose capability table lists managed worktrees as deferred.

## Where to fix

- `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-wrap-epic/SKILL.md`
  — the epic's worktree is the operator's to time, and the branch stays; say
  both, and say they are separate decisions.
- `internal/skills/embedded/aiwf-worktree/SKILL.md` — why `add` has no inverse.
- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`
  and `.../aiwfx-wrap-milestone/SKILL.md` — the two that already dispose, if the
  duplicated resolution block is worth reconciling once the boundary is written.
  It may not be: each removes a worktree its own ritual created, which is the
  case the boundary permits.
