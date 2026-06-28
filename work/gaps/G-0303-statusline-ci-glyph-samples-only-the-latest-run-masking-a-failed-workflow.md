---
id: G-0303
title: statusline CI glyph samples only the latest run, masking a failed workflow
status: addressed
addressed_by_commit:
    - 424bdc83
---
## Problem

The statusline's CI segment (`.claude/statusline.sh`) resolves the CI glyph by
running `gh run list --branch <b> --limit 1` and reading `.[0]` — the single
most-recent run. When a push fires multiple workflows and one fails (e.g. the
`go` test workflow) while others pass (link-check, markdown-lint, scrub,
gitleaks), the latest run is a *passing* one, so the glyph shows a green ✓ while
CI is actually red. Observed live on the E-0047 wrap push (commit efd11d94): the
`go` workflow failed the coverage gate, but the statusline showed ✓ from the
link-check run.

## Direction

Aggregate across all workflows for the checked-out HEAD instead of sampling
`.[0]`: fetch the runs (`gh run list --branch <b> --limit N --json
conclusion,status,headSha`), filter to the HEAD sha, and reduce to the worst
state — any failure/cancelled/timed_out yields ✗; else any in_progress/queued
yields the running glyph; else all success yields ✓. Preserve the existing
HEAD-staleness guard and the main-fallback behavior. Surfaced while shipping
E-0047.
