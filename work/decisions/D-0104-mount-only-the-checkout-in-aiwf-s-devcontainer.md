---
id: D-0104
title: Mount only the checkout in aiwf's devcontainer
status: proposed
relates_to:
    - G-0524
    - E-0098
    - M-0359
---
> **Date:** 2026-09-29 · **Decided by:** Peter Bruinsma

## Question

aiwf's hand-written devcontainer bound the parent of the clone, so every sibling repository
beside it, and any instruction file at that level, was visible and writable inside the container.
G-0524 records the cost: a parent-level `CLAUDE.md` displaced this repository's own instructions
in a live session, and every agent session had write reach across dozens of unrelated repositories.
The same bind was a documented capability: it let the container reach a sibling repository, or
open directly on a sibling worktree. Moving the container onto the house devcontainer kit settles
whether that reach survives, and in what form.

## Decision

- aiwf's devcontainer mounts only the checkout: `workspaceMount` binds `${localWorkspaceFolder}`
  at `/workspaces/aiwf`, and no mount binds its parent.
- A sibling repository the container needs is mounted explicitly, through the kit's `siblings`
  (read-only) or `writable_siblings` answers, added with `uvx copier update --data`.
- None is listed now.
- The sibling-worktree `.git` rewrite goes with the parent bind.

## Reasoning

- **Explicit sibling mounts over the parent bind.** A listed sibling is visible in the
  configuration and narrow; the parent bind exposed everything beside the clone, including files
  no one chose to share, and nothing in aiwf's own discipline reaches past this repository.
- **No siblings for now.** The work that used the reach, setting up other repositories from this
  container, happens in each repository's own container, which carries the same tools. A sibling
  is one answer away when a need recurs.
- **The `.git` rewrite goes.** It rewrote a sibling worktree's `gitdir` to a relative path that
  resolved only because siblings shared the mounted parent; aiwf places worktrees inside the
  repository, where no rewrite is needed.

## Consequences

- `PolicyDevcontainerWorkspaceMount` (`internal/policies/devcontainer_workspace_mount.go`) fails
  when `workspaceMount` or any mount binds the checkout's parent.
- Cross-repository work that ran from this container moves to the other repository's container,
  or to a sibling answer added for it.
- `.devcontainer/project/README.md` states how to add a sibling.
