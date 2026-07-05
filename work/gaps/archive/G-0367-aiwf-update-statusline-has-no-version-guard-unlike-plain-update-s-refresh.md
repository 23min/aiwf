---
id: G-0367
title: aiwf update --statusline has no version guard unlike plain update's refresh
status: addressed
addressed_by_commit:
    - 5df2aa0b
---
## What's missing

`aiwf update --statusline` (`ScaffoldStatuslineWithHome`, `internal/skills/statusline.go`) always renders to the running binary's embedded copy and writes it whenever it differs from what's on disk — with no `version.Compare` check at all. This is deliberate for the case of an operator on a real installed release forcing a refresh, but it applies identically to a worktree/dev-built binary of this repo: a `go build` binary carries a Stamp like `<branch>@<short-sha>[-dirty]` (the Makefile's `AIWF_VERSION`), which is untagged and therefore unorderable — yet the explicit `--statusline` path writes unconditionally regardless. Contrast with plain `aiwf update`'s auto-refresh (`AutoRefreshStatuslineForVersion` / `decideStatuslineRefresh`), which is upgrade-only and skips whenever `version.Compare` returns `SkewUnknown`. Since `--scope user` is the default and shared across every repo/container under $HOME, running `--statusline` from a worktree build silently pushes unreleased/experimental script content live everywhere that shares that home.

## Why it matters

An operator diagnosing aiwf's own behavior with a worktree-scoped binary (per this repo's own worktree binary discipline guidance) could reasonably reach for `--statusline` to force-refresh, not realizing the write is unconditional and scope defaults to user. The result: every other project's Claude Code session sharing that $HOME starts running unreleased, possibly-buggy statusline logic, with no version marker distinguishing it as non-release. The blast radius is bounded (the statusline is advisory UI text, not planning-tree state), but the guard gap is real: the one destructive, shared-scope write path in the whole statusline surface is exactly the one with no ordering check. A fix likely gates the explicit write behind a confirmation (or requires an explicit override) when the running binary's version is untagged/unorderable, so a dev build needs a deliberate act before landing in the shared user-scope file.