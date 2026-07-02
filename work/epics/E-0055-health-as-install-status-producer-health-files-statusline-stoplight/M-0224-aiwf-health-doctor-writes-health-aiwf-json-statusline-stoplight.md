---
id: M-0224
title: 'aiwf health: doctor writes health.aiwf.json + statusline stoplight'
status: in_progress
parent: E-0055
tdd: required
acs:
    - id: AC-1
      title: doctor exposes its warnings and errors with severity and message
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf writes .claude/health.aiwf.json from doctor's warnings and errors
      status: open
      tdd_phase: done
    - id: AC-3
      title: statusline reads the health files and renders the four-state stoplight
      status: open
      tdd_phase: red
---
## Deliverable

Give operators visibility of `aiwf`'s installation and configuration warnings and errors
in the statusline. Two halves, landed as one milestone:

- **Producer** — surface `aiwf doctor`'s warnings and errors as structured problems and
  write them to `.claude/health.aiwf.json` (the fixed ai-dotfiles schema).
- **Consumer** — the statusline reads `.claude/health.*.json`, unions across producers,
  and renders a four-state stoplight at the maximum severity, never running a check on the
  render path.

## Acceptance criteria

### AC-1 — doctor exposes its warnings and errors with severity and message

`aiwf doctor` collects its problem states — the blocking checks (today's error count) and
the advisory ones — as `{severity, message}`, `severity` one of `warn` / `error`. The
existing human report is unchanged. Evidence: a repo with a known problem (e.g. a missing
`aiwf.yaml`) yields an error-severity problem whose message names it; a clean section
yields none.

### AC-2 — aiwf writes .claude/health.aiwf.json from doctor's warnings and errors

`aiwf doctor --write-health` maps those problems onto the fixed ai-dotfiles schema
(`{generated_at, findings:[{source:"aiwf", severity, message}]}`; empty `findings` when
healthy) and atomic-writes the file to the main checkout's `.claude/`, resolved even from a
linked worktree; `aiwf update` writes it too. Evidence: seam tests driving `doctor
--write-health` and `aiwf update`; a linked-worktree resolution test; the healthy →
empty-findings mapping.

### AC-3 — statusline reads the health files and renders the four-state stoplight

The statusline globs `.claude/health.*.json`, unions the findings, and prefixes the line
with a four-state stoplight at the maximum severity: green `●` healthy, yellow `▲` warn,
red `▲` error, gray `●` unknown (no readable file). It runs no check on the render path.
Evidence: behavioral tests per state, the cross-producer union (max severity wins), and the
all-corrupt → gray degrade.
