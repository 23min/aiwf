---
id: G-0095
title: internal/policies/ walker doesn't exclude .claude/ — false-positive flags from sibling worktree .go files
status: addressed
addressed_by_commit:
    - f3810cb
---
## Problem

`internal/policies/policies.go::WalkGoFiles` walks the kernel repo for production `.go` files, skipping `vendor`, `node_modules`, and `.git`. It does **not** skip `.claude/`.

Claude Code's worktree directories live at `.claude/worktrees/agent-*/` and contain a full clone of the kernel source tree. When a developer or AI agent has an active worktree, every policy that walks Go files (`trailer-keys-via-constants`, `no-history-rewrites`, `no-timestamp-manipulation`, `no-signature-bypass`, `closed-set-status-via-constants`) re-flags the worktree's intentional definitions of trailer literals, `--force` mentions, `GIT_AUTHOR_DATE` references, and the like as production violations of the kernel itself.

The flags are *correct in form* (the literal strings exist) but *wrong in target* — those files are the legitimate definitions inside a sibling clone. The pre-commit `.local` hook (which runs `go test ./internal/policies/...`) blocks every commit attempt while a worktree is active.

## Fix

Add `.claude` to the directory-name skip list in `WalkGoFiles` alongside `vendor`, `node_modules`, `.git`. One-line change plus a focused test (`TestWalkGoFiles_SkipsExcludedDirs`) that pins the exclusion list against drift.

## Discovered

While running the wf-patch that closes G-0057 + G-0086. The user had an active worktree at `.claude/worktrees/agent-ae85cfc7baf987101/`; the policy gate started failing the moment the verb tried to commit.
