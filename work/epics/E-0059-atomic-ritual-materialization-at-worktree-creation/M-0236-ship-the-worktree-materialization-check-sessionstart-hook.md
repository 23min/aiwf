---
id: M-0236
title: Ship the worktree-materialization-check SessionStart hook
status: draft
parent: E-0059
depends_on:
    - M-0235
tdd: required
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

## Constraints

- Detection is harness-executed only — no skill-instruction fallback; the
  whole point is removing the LLM-memory dependency.
- Advisory only — never hard-refuses, per the epic's own risk mitigation
  against false positives on an intentionally bare worktree.

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
