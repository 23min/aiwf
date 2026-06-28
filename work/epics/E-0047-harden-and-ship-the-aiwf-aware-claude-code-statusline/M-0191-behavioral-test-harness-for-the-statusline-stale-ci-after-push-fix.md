---
id: M-0191
title: Behavioral test harness for the statusline + stale-CI-after-push fix
status: draft
parent: E-0047
tdd: required
acs:
    - id: AC-1
      title: Behavioral test runs statusline.sh against a fixture and asserts real output
      status: open
      tdd_phase: red
    - id: AC-2
      title: CI segment shows pending when the run's headSha differs from local HEAD
      status: open
      tdd_phase: red
    - id: AC-3
      title: Statusline cache key includes HEAD sha so a push invalidates a stale CI result
      status: open
      tdd_phase: red
---
## Deliverable

A behavioral test harness for `.claude/statusline.sh` (G-0187), plus the stale-CI-after-push fix (G-0189) as its first target.

**Harness (G-0187).** A Go test that writes a known-shape transcript fixture + a temp git repo, streams a stub stdin JSON through `exec.Command("bash", scriptPath)`, strips ANSI from the rendered output, and asserts the *segment shapes* from real output (token count, sync ahead/behind, CI segment). This replaces the regex-over-source assertions in `internal/policies/statusline_content_test.go` (which never run the script — the `||` binding bug nearly shipped because of exactly that) with assertions that exercise behavior.

**Stale-CI fix (G-0189).** The CI segment compares the latest run's `headSha` against local `git rev-parse HEAD`; on mismatch it renders `… ci` (gray, pending) instead of the previous run's stale `✓`. HEAD is folded into the cache key so a push auto-invalidates.

## Why combined (per the epic)

The harness proves itself by catching and fixing the clearest statusline bug; the stale-CI fix is its first behavioral target. Every later milestone (M2–M4) asserts against this harness.

### AC-1 — Behavioral test runs statusline.sh against a fixture and asserts real output

### AC-2 — CI segment shows pending when the run's headSha differs from local HEAD

### AC-3 — Statusline cache key includes HEAD sha so a push invalidates a stale CI result

