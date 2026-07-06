---
id: M-0235
title: 'Generalized hook registry: aiwf.yaml-declared, persisted consent'
status: draft
parent: E-0059
tdd: required
---

## Goal

Add a harness-executed hook that detects a session or subagent starting with cwd
inside an un-materialized `.claude/worktrees/` checkout, and warns before
proceeding — a backstop for any worktree created outside the M-0233 wrapper.

## Context

M-0233/M-0234 close the aiwf-initiated creation path; this milestone catches
everything else — a bare `git worktree add` run directly, or any path this epic
doesn't rewire. Modeled on the existing `.claude/hooks/validate-agent-isolation.sh`
PreToolUse hook pattern from G-0099, which already demonstrates a harness-executed
(not skill-instruction) hook enforcing a worktree-related invariant in this repo.

## Acceptance criteria

<!-- ACs allocated at aiwfx-start-milestone via `aiwf add ac M-0235 --title "..."`.
     Candidate AC titles, drafted here as prose hints (not yet kernel state): -->

- **AC-1 candidate** — Resolve the open question from the E-0059 epic spec: spike
  whether a Claude Code `SessionStart` hook can actually block/warn before the
  session proceeds, or only inject context after the fact. Record the finding and
  size the remaining ACs against whichever mechanism is actually available.
- **AC-2 candidate** — The hook fires when cwd is under `.claude/worktrees/` and
  `.claude/skills/` is absent or stale (not byte-equal to the embed), and stays
  silent otherwise (main checkout, a fully-materialized worktree).
- **AC-3 candidate** — The hook is advisory (warns) by default, not a hard
  refusal — per the epic's risk mitigation against false positives on an
  intentionally-bare worktree (e.g. a throwaway checkout for unrelated
  inspection).
- **AC-4 candidate** — Contract pinned by a `Test...` under `internal/policies/`,
  matching the `TestAgentIsolationHook_*` precedent from G-0099.

## Constraints

- The hook is harness-executed (materialized via `aiwf init`/`update`), not a
  skill instruction — removing the LLM-memory dependency is the entire point; the
  check itself cannot be another piece of advisory prose.
- Scoped strictly to `.claude/worktrees/` (the aiwf-owned convention per
  ADR-0023), never a general cwd check.

## Out of scope

- The verb (M-0233) and its ritual rewiring (M-0234) — this is the detection-only
  backstop, not the structural-creation fix.
- Retrofitting already-existing bare worktrees (excluded at the epic level).

## Dependencies

- None within this epic — independent of M-0233/M-0234; may ship in parallel with
  either.

## References

- G-0374 — the gap this epic closes.
- G-0099 — the adjacent isolation-as-precondition concern; `.claude/hooks/validate-
  agent-isolation.sh` and `TestAgentIsolationHook_*` — the existing hook pattern
  this milestone follows.
