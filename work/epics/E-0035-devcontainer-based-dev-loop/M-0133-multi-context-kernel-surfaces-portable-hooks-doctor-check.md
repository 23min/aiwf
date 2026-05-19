---
id: M-0133
title: 'Multi-context kernel surfaces: portable hooks + doctor check'
status: draft
parent: E-0035
tdd: required
---
## Goal

Eliminate the three kernel surfaces that assume single-context dev
and bite every multi-context session (host ↔ devcontainer ↔
worktree). The "multi-context tax" — hook-path tug-of-war, inert
worktree hook writes, and false-positive plugin warnings — currently
forces operators to remember per-context discipline and re-run
`aiwf init` / `aiwf update` when switching environments, or
explicitly ignore warnings that are known-false. Land three surgical
fixes so the same checkout works frictionlessly across all contexts
after one `aiwf init`.

## Approach

Three small, surgical kernel-side changes, each closing one gap,
shipped as one milestone so the multi-context dev story lands as
one coherent improvement.

1. **Portable hook binary lookup** (closes [G-0135](../../../gaps/G-0135-hook-path-hardcoded-at-install-time-breaks-across-gopath-environments.md)).
   Replace the install-time absolute `aiwf` path baked into
   `pre-commit` / `pre-push` / `post-commit` hooks with a PATH-relative
   `command -v aiwf` resolution at hook execution time. Same hook
   contents work from any environment whose PATH contains `aiwf`.
   Touches the hook-installation template (in `internal/cli/install/`
   or wherever the hook generator lives; verified on first read).
   Matches the design-decisions §"Contracts" posture that the
   engine doesn't ship binaries — PATH lookup is the consumer's job.

2. **`aiwf update` writes to shared hooks dir from worktrees**
   (closes [G-0136](../../../gaps/G-0136-aiwf-update-from-a-worktree-writes-hooks-git-ignores.md)).
   When `aiwf update` runs in a worktree, resolve the hooks directory
   via `git rev-parse --git-common-dir` so the write lands at the
   main repo's shared `.git/hooks/`, not the inert per-worktree
   `.git/worktrees/<id>/hooks/`. Per-worktree hook divergence is
   not a current use case; the shared-write model matches the actual
   semantic (aiwf hooks are repo-level policy). Operator output
   explicitly states the write affects all worktrees.

3. **`aiwf doctor` recommended-plugins reads `enabledPlugins`**
   (closes [G-0138](../../../gaps/G-0138-aiwf-doctor-recommended-plugin-check-false-positives-across-multi-context-dev.md)).
   Switch the recommended-plugins check's source of truth from
   machine-local `~/.claude/plugins/installed_plugins.json`
   (path-strict; false-positives across worktrees, containers, and
   re-clones) to `<rootDir>/.claude/settings.json`'s `enabledPlugins`
   map (in the source tree; path-independent by construction). Drops
   the path-equality comparison entirely. Secondary: fix the
   install-advice string to include `--scope project` (matches the
   CLAUDE.md operator-setup recipe; the bare CLI form defaults to
   user-scope per Claude Code docs).

Each change carries its own AC with a mechanical assertion under
`internal/policies/`. The three changes touch independent code paths
(hook template / install verb / doctor verb), so implementation order
is flexible.

## Acceptance criteria

ACs land via `aiwf add ac M-NNNN`; the three are summarised above.
