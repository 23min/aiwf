---
id: M-0188
title: Pin that the loader ignores in-repo worktrees under .claude/worktrees
status: in_progress
parent: E-0046
tdd: none
---

# M-0188 — Pin that the loader ignores in-repo worktrees under .claude/worktrees

## Goal

Pin, with a regression test, that the aiwf loader / `aiwf check` does not descend into
in-repo worktrees under `.claude/worktrees/` — so a nested second checkout there cannot
surface phantom duplicate entities once in-repo worktrees become the default placement.

## Acceptance criteria

_Scaffolded via `aiwf add ac` at start-milestone. Intended shape: (1) a fixture
planning-tree containing a full second checkout under `.claude/worktrees/<branch>/` (with
its own `work/...`) yields zero phantom or duplicate-entity findings from `aiwf check`;
(2) the test goes red if the loader is altered to descend into `.claude/worktrees/` — a
vacuity guard proving the assertion can catch the regression._

## Context

The epic (E-0046) makes in-repo worktrees the default placement. An in-repo worktree is a
full second checkout of the repo *inside the tree*, including its own `work/...`. If the
loader walked from the repo root into `.claude/worktrees/`, it would load duplicate entity
files and report false id collisions. The behavior is likely already correct (`.claude/*`
is gitignored; the loader reads `work/`/`docs/`), so this milestone verifies first, then
pins the result — it must not remain an assumption.

## Constraints

- Pins behavior, not implementation: asserts `aiwf check` output on a fixture, with a
  vacuity check that the assertion fails when the guard is removed.
- Resolve entity paths via the loader, never hardcoded (CLAUDE.md "Policy tests … resolve
  via the loader").

## Out of scope

- The config knob (M-0189) and the ritual default (M-0190) — this milestone only guards
  the loader.

## Dependencies

- None. Sequenced first to de-risk the default flip.

## References

- E-0046 epic spec; CLAUDE.md "Subagent worktree isolation".
