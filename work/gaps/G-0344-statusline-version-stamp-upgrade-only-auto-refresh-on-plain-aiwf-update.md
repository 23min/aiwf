---
id: G-0344
title: 'statusline: version-stamp + upgrade-only auto-refresh on plain aiwf update'
status: open
---
## Problem

For most operators `~/.claude` is a shared host↔container bind mount (the codebase
already acknowledges this), so a user-scope statusline (`$HOME/.claude/statusline.sh`)
is effectively **central config shared across every container and the host**. That is
often *desirable* — one statusline everyone gets. But the shared artifact is not
governed for that reality:

1. **No auto-refresh on plain `aiwf update`.** The materialized statusline refreshes only
   when `--statusline` is re-passed; a plain `aiwf update` (or `aiwf upgrade`) leaves it
   untouched — unlike skills, hooks, and the guidance fragment, which always refresh. So
   it silently goes stale, and the operator must remember the flag for "aiwf to just keep
   it current."

2. **No version stamp → unsafe under sharing.** "Current" is inferred by byte-comparing
   the on-disk script to the running binary's embed. With one mutable script shared across
   containers that may run *different* aiwf versions, a byte-diff cannot distinguish
   "newer" from "older": an older-version container's refresh would silently **downgrade**
   the shared script a newer-version container installed (last-writer-wins thrash).

The net effect surfaced when the statusline health glyph (E-0055) appeared live across
all of an operator's devcontainers at once. Analysis of the scope options concluded that
switching to `--scope project` is the wrong fix (it is worktree-fragile — G-0337 — and
aiwf's own rituals run in worktrees), and containerizing `~/.claude` sacrifices the
central-config the operator wants. The right model is a **well-governed host-shared
statusline**, which needs the two properties below.

## Direction

1. **Version-stamp the embedded statusline** — a marker/version header, in the spirit of
   the `# aiwf:<hook>` hook markers and the guidance fragment's stamp — so aiwf can
   recognize its own managed copy and read its version without a byte-compare.

2. **Upgrade-only auto-refresh on plain `aiwf update`.** When an aiwf-stamped statusline
   is already installed, refresh it in place on the normal `aiwf update` — but **only when
   the binary's embed version is newer-or-equal** to the installed stamp (never a blind
   downgrade). This rewrites only the gitignored script file; `settings.json` is already
   wired, so no `settings.json` edit occurs and the ADR-0015 consent gate is not
   re-triggered.

3. **`aiwf doctor` reports installed-vs-embed statusline version** — a version-aware form
   of today's byte-diff drift warning, so "which version is live, and is it current?" is
   answerable at a glance across a fleet of containers.

4. **Initial install + `settings.json` wiring stay behind `--statusline`** (ADR-0015
   consent unchanged). Only the script-content refresh of an *already-installed* copy
   rides plain `aiwf update`.

Relates to G-0312 (materialized statusline refresh — addressed for the `--statusline`
path; this extends it to plain `aiwf update` plus versioning) and G-0337 (statusline
scope robustness / worktree fragility, which is why user scope, not project scope, is the
form being governed here).
