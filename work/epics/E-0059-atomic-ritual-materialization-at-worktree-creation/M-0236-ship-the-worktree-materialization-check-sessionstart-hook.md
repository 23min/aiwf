---
id: M-0236
title: Ship the worktree-materialization-check SessionStart hook
status: in_progress
parent: E-0059
depends_on:
    - M-0235
tdd: required
acs:
    - id: AC-1
      title: Hook flags unmaterialized worktree rituals, nonzero exit with stderr
      status: open
      tdd_phase: green
    - id: AC-2
      title: Hook registered in the registry for both SessionStart and SubagentStart events
      status: open
      tdd_phase: red
    - id: AC-3
      title: Subprocess policy test pins exit code and stderr for both hook cases
      status: open
      tdd_phase: red
    - id: AC-4
      title: init/update materialize and wire the hook per consent; doctor reports its state
      status: open
      tdd_phase: red
---

## Goal

Ship the one concrete hook this epic exists to add: a `SessionStart` /
`SubagentStart` script that detects a session or subagent starting with cwd
inside a `.claude/worktrees/` checkout whose rituals aren't materialized,
and warns visibly without blocking — registered against the hook registry
the prior milestone builds.

## Context

M-0235 lands the generalized hook registry: the `aiwf.yaml` `hooks:` consent
schema, the settings.json writer, and the materialization category. This
milestone adds the first (and, for now, only) entry in that registry.

Spike finding (resolved this session, against the official Claude Code
hooks documentation): `SessionStart` and `SubagentStart` cannot block or
abort — their only user-visible channel is a nonzero exit code, whose
stderr renders as a harness-level "hook error notice" directly to the
human, without the model mediating it, while the session or subagent
proceeds regardless. That is exactly the advisory, never-hard-refuse shape
this epic's own risk mitigation calls for, and it sidesteps relying on the
LLM to relay a silently-injected `additionalContext` string — the harness
renders the notice itself.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0236 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — The hook script checks whether cwd is under
  `.claude/worktrees/`; if so, it checks ritual materialization (reusing
  `aiwf doctor`'s existing rituals-presence check rather than
  reimplementing it) and exits nonzero with a clear, actionable stderr
  message when rituals are missing or stale. It exits 0 silently for the
  main checkout, or a worktree whose rituals are fully materialized.
- **AC-2 candidate** — The script is registered in the hook registry under
  both `SessionStart` and `SubagentStart`, so it fires for an interactively
  started session and for a dispatched subagent alike.
- **AC-3 candidate** — Contract pinned by a subprocess-level policy test
  mirroring `TestAgentIsolationHook_*` (`internal/policies/`), asserting
  exit code and stderr content for both the stale/missing case and the
  healthy case.
- **AC-4 candidate** — `aiwf init` / `aiwf update` materialize the script and
  wire both settings-json event arrays once the operator has consented via
  M-0235's registry mechanism; `aiwf doctor` reports the hook's
  materialized/wired state.

### AC-1 — Hook flags unmaterialized worktree rituals, nonzero exit with stderr

### AC-2 — Hook registered in the registry for both SessionStart and SubagentStart events

### AC-3 — Subprocess policy test pins exit code and stderr for both hook cases

### AC-4 — init/update materialize and wire the hook per consent; doctor reports its state

## Constraints

- Detection is harness-executed only — no skill-instruction fallback; the
  whole point is removing the LLM-memory dependency.
- Advisory only — never hard-refuses, per the epic's own risk mitigation
  against false positives on an intentionally bare worktree.
- Ships complete, standalone `--help` documentation for this concrete
  hook: replace M-0235's placeholder `aiwf init --enable-hook <hook-name>`
  / `aiwf update --enable-hook <hook-name>` Example lines with this hook's
  real name, and state plainly what enabling it does. `--help` is the
  shippable discovery channel for `init`/`update` (both allowlisted
  no-skill ops verbs per ADR-0006) — no CLAUDE.md mention needed, and no
  reference to any sibling consent mechanism (ADR-0015/ADR-0018).
- The settings.json command this milestone wires via `WireHookSettings`
  must be exactly `<Target.HooksDir>/<hook-name>` (e.g.
  `.claude/hooks/<hook-name>`) — `aiwf doctor`'s hook-drift check
  (`skills.HookDrift`, M-0235/AC-5) detects "wired" by matching that exact
  string against every command in `settings.json`'s `hooks:` key. A
  different command shape (an env-var prefix, an absolute path) silently
  breaks the drift report without erroring — `HookDrift`'s derivation must
  be revisited if this milestone needs a different convention.
- `aiwf doctor`'s "wired-but-stale" hook report (`skills.HookDrift`,
  M-0235/AC-5) checks script presence only (`os.Stat`), not content — it
  cannot detect an on-disk script whose bytes no longer match the shipped
  `HookDef.Content`. Confirm presence-only is the intended scope for this
  hook, or extend `HookDrift` with a content comparison if staleness needs
  to mean "outdated bytes," not just "decision drift."

## Out of scope

- The hook registry mechanism itself, `aiwf.yaml`'s `hooks:` schema, and the
  settings.json writer — M-0235.
- Migrating `.claude/hooks/validate-agent-isolation.sh` (G-0099) into the
  registry — a follow-up gap.

## Dependencies

- M-0235 (hook registry) — this milestone registers its one concrete hook
  against that infrastructure and cannot start before it lands.

## References

- G-0374 — the gap this epic closes.
- G-0099 — the sibling isolation-guard hook; its migration into this
  registry is a follow-up gap, not this milestone's work.
