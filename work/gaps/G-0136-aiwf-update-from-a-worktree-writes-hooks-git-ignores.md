---
id: G-0136
title: aiwf update from a worktree writes hooks git ignores
status: addressed
addressed_by:
    - M-0133
---
## Problem

When `aiwf update` runs inside a git worktree (cwd is the worktree, not the main checkout), it materializes hook files at `<main-repo>/.git/worktrees/<id>/hooks/` — but git's default hook lookup consults the SHARED `<main-repo>/.git/hooks/` directory, not `.git/worktrees/<id>/hooks/`.

Empirically verified on the git version in use here: hooks written to the per-worktree path are inert. Only updating the shared `.git/hooks/` (by running `aiwf update` from the main checkout) actually changes hook behavior.

`aiwf update`'s output suggests the worktree hooks are "created" and functional, which is misleading. A user troubleshooting a hook problem from inside a worktree may run `aiwf update`, see "created" messages, and assume the issue is fixed — when in fact the shared hook hasn't been touched.

## Surfaced via

E-0033 / M-0123 / phase 2 AC-2 commit. Initial `aiwf update` from the worktree wrote worktree-specific hooks; the next git commit attempt still ran the shared (broken-path) hook. A second `aiwf update` from the MAIN checkout fixed the shared hook.

## Proposed fix shape

Three candidates:

1. **Skip worktree-specific writes** when `aiwf update` detects it's running in a worktree. Either always write to the shared hooks directory (resolved via `git rev-parse --git-common-dir`), or refuse and tell the user to run from the main checkout.
2. **Set `core.hooksPath`** on the worktree's local config to point at the worktree-specific hooks directory. Git WILL consult that path when set. This makes per-worktree hooks meaningful and respects worktree isolation.
3. **Document only**: keep the current behavior but make `aiwf update`'s output explicit about worktree-vs-shared and recommend running from main for shared updates.

Option 1 is simplest if per-worktree hook divergence isn't a real need. Option 2 enables per-worktree hook flexibility that may be useful for the upcoming devcontainer story (E-0035).

## Related

- The hook-path-hardcoded gap (filed alongside) — same surface, different facet. Could be merged into one "aiwf update hook materialization is environment-fragile" umbrella; kept separate per the user's option-1 selection in M-0123's mid-flight gap-filing decision.

## Discipline today

Run `aiwf update` from the main checkout (cwd = repo root, not the worktree) when shared-hook changes are needed.
