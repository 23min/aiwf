---
id: G-0436
title: CLAUDE.md and id-allocation.md cite stale cmd/aiwf/ paths for relocated verbs
status: open
discovered_in: E-0034
---
## What's missing

Two normative docs cite `cmd/aiwf/` source paths for files that no longer live there — an unrelated `cmd/aiwf/` -> `internal/cli/` source refactor moved the underlying code, but the doc citations weren't updated:

- `CLAUDE.md` "What's enforced and where" cites `cmd/aiwf/completion_drift_test.go`; the file is now `internal/cli/integration/completion_drift_test.go`.
- `docs/design/id-allocation.md` cites `cmd/aiwf/admin_cmd.go` for `runHistory`/`readHistoryChain`; the file no longer exists under `cmd/aiwf/` (only `main.go` remains there).

## Why it matters

A reader following either citation to understand the kernel's structure lands on a nonexistent path. Surfaced by an epic-level doc-lint sweep during E-0034's wrap (a full repo-wide link-integrity pass); confirmed pre-existing via `git show main:CLAUDE.md` (byte-identical before E-0034) and pre-dating `docs/design/id-allocation.md`'s relocation from `docs/pocv3/design/` (the citation was already stale at its old path). Orthogonal to E-0034's scope (documentation-tree retirement), so filed here rather than fixed inline.

## Closing this gap

Update both citations to their current `internal/cli/` paths. Small, mechanical fix — no design work needed.