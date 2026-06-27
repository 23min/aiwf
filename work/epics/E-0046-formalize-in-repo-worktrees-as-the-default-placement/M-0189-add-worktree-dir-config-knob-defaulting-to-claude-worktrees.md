---
id: M-0189
title: Add worktree.dir config knob defaulting to .claude/worktrees
status: in_progress
parent: E-0046
tdd: required
acs:
    - id: AC-1
      title: aiwf.yaml worktree.dir is parsed and exposed through config
      status: open
      tdd_phase: red
    - id: AC-2
      title: Unset, empty, or invalid worktree.dir defaults to .claude/worktrees
      status: open
      tdd_phase: red
    - id: AC-3
      title: aiwf doctor surfaces the resolved worktree.dir line
      status: open
      tdd_phase: red
---

# M-0189 — Add worktree.dir config knob defaulting to .claude/worktrees

## Goal

Add a `worktree.dir` key to `aiwf.yaml` (default `.claude/worktrees`) giving a project a
persistent default placement for ritual worktrees, surfaced where the start rituals can
read it.

## Acceptance criteria

_Scaffolded via `aiwf add ac` at start-milestone (tdd: required — ACs seed at red).
Intended shape: (1) `aiwf.yaml worktree.dir` is parsed and exposed through the config
surface; (2) an unset or empty value defaults to `.claude/worktrees`; (3) the resolved
value is reachable by the rituals (exact surface — e.g. an `aiwf doctor` line or a
config-get path — decided at start-milestone)._

## Context

`aiwf.yaml` already carries top-level feature keys (`tree:`); a `worktree:` key follows
the established shape. The correct placement is environment-dependent (in-repo for
sandboxed devcontainers, siblings on a bare host), so the default is config-driven, not
hardcoded (E-0046 constraint). This milestone adds only the knob + default + parse; the
rituals consume it in M-0190.

## Constraints

- Minimal surface: a single repo-relative directory value (YAGNI — no absolute paths or
  multiple roots until a consumer needs them).
- Unset / empty / invalid values fall back to the kernel default `.claude/worktrees`.

## Out of scope

- The rituals reading/defaulting to the knob (M-0190); the loader guard (M-0188).

## Dependencies

- None.

## References

- E-0046 epic spec; `aiwf.yaml` `tree:` key precedent.

### AC-1 — aiwf.yaml worktree.dir is parsed and exposed through config

### AC-2 — Unset, empty, or invalid worktree.dir defaults to .claude/worktrees

### AC-3 — aiwf doctor surfaces the resolved worktree.dir line

