---
id: M-0224
title: doctor surfaces warn/error and writes health.aiwf.json
status: in_progress
parent: E-0055
tdd: required
acs:
    - id: AC-1
      title: doctor exposes its warnings and errors with severity and message
      status: open
      tdd_phase: red
    - id: AC-2
      title: aiwf writes .claude/health.aiwf.json from doctor's warnings and errors
      status: open
      tdd_phase: red
    - id: AC-3
      title: statusline reads the health files and renders the four-state stoplight
      status: open
      tdd_phase: red
---
## Deliverable

Surface `aiwf doctor`'s warnings and errors as structured problems and write them to
`.claude/health.aiwf.json` for the statusline (and any other consumer) to read. The
problems are collected alongside the existing doctor output — no rewrite of the report.

## Acceptance criteria

### AC-1 — doctor exposes its warnings and errors with severity and message

`aiwf doctor` collects its problem states — the blocking checks (today's error count)
and the advisory ones — as `{severity, message}`, `severity` one of `warn` / `error`
(`info` reserved for non-actionable context). The existing human report is unchanged.
Evidence: a repo with a known problem (e.g. a missing `aiwf.yaml`) yields an
error-severity problem whose message names it; a clean repo yields none.

### AC-2 — aiwf writes .claude/health.aiwf.json from doctor's warnings and errors

`aiwf doctor --write-health` maps those problems onto the fixed ai-dotfiles schema
(`{generated_at, findings:[{source:"aiwf", severity, message}]}`; empty `findings` when
healthy) and atomic-writes the file to the main checkout's `.claude/`, resolved even
from a linked worktree; `aiwf update` writes it too, so it refreshes on the command
operators already run. Evidence: a seeded-problem repo produces the mapped error entry;
a healthy repo produces an empty `findings` array; the write is atomic.

### AC-3 — statusline reads the health files and renders the four-state stoplight

