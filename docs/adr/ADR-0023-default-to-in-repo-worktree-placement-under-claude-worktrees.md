---
id: ADR-0023
title: Default to in-repo worktree placement under .claude/worktrees
status: accepted
---

# ADR-0023 — Default to in-repo worktree placement under .claude/worktrees

> **Date:** 2026-06-27 · **Decided by:** human/peter

## Context

aiwf's start rituals (`aiwfx-start-epic`, `aiwfx-start-milestone`) create a git worktree
for implementation work, and historically offered three placements as a free choice with
no default: switch branches in the main checkout, an in-repo worktree under
`.claude/worktrees/`, or a sibling directory next to the repo (the near-universal git
convention).

The forcing function is the **devcontainer sandbox**. A Claude Code session running in a
devcontainer is confined to the workspace folder. This was established empirically:

- A `cd` into a **sibling** worktree (outside the workspace, e.g.
  `/workspaces/<repo>-<branch>`) is **reset** back to the workspace root — the session can
  never root there.
- A `cd` into an **in-repo** worktree (a subdirectory of the workspace) **persists**, and
  the statusline correctly follows the session into it, with Claude Code populating the
  `workspace.git_worktree` field.

So the alternatives carry concrete costs in a container:

- **Sibling worktrees are unreachable** as a session's working directory. Surfaces that
  derive context from the cwd — the statusline's active entity, branch, CI — can never
  reflect the work, because the session is pinned to the workspace root.
- **`$HOME`-placed worktrees are wiped on container rebuild.** `$HOME` is typically not a
  persistent mount, so worktrees there (and any un-pushed commits) are a real lost-work
  hazard.
- **In-repo worktrees are reachable, persistent, and gitignored** (`.claude/*`). They live
  under the mounted workspace, so they neither clutter the parent directory nor depend on
  an ephemeral mount.

The correct placement is environment-dependent: in-repo is necessary in a sandboxed
devcontainer; siblings remain reasonable on a bare host. So the choice is a *default*, not
a hardcode.

## Decision

The kernel default placement for worktrees created by the start rituals is
`.claude/worktrees/<branch>/` (in-repo). The default is overridable per-project via
`aiwf.yaml worktree.dir`, and the per-invocation placement choice (main checkout, sibling)
remains available in the ritual Q&A. The knob sets the default; it does not lock placement.

This generalizes the existing CLAUDE.md "Subagent worktree isolation" convention — which
already names `.claude/worktrees/<name>` for transient agent worktrees — to the documented
default for session and epic worktrees.

## Consequences

- The start rituals default to in-repo placement, reading the knob.
- A `worktree.dir` key is added to `aiwf.yaml`, defaulting to `.claude/worktrees`.
- **The loader / `aiwf check` must not descend into `.claude/worktrees/`.** An in-repo
  worktree is a full second checkout of the repo *inside the tree*, including its own
  `work/...`; if the loader walked into it, it would load duplicate entity files and report
  phantom id collisions. This invariant is pinned by a regression test.
- Siblings remain valid on a bare host via the knob or the per-invocation override; no
  behavior is removed — only the default changes.
- A second checkout under the repo root means tooling that recurses from the root and
  ignores `.gitignore` (`grep -r`, `find`) will see duplicates; gitignore-aware tools
  (ripgrep, `git grep`, Go's `./...`) skip it.

## References

- Epic: E-0046 — Formalize in-repo worktrees as the default placement.
- Milestones: M-0188 (loader guard), M-0189 (`worktree.dir` knob), M-0190 (ritual default).
- CLAUDE.md — "Subagent worktree isolation", "Worktree binary discipline".
