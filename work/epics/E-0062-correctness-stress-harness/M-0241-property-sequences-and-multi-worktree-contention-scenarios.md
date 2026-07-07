---
id: M-0241
title: Property sequences and multi-worktree contention scenarios
status: draft
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: Generated verb sequences are checked for FSM legality each step
      status: open
      tdd_phase: red
    - id: AC-2
      title: Real subprocesses racing repolock never produce a duplicate id
      status: open
      tdd_phase: red
    - id: AC-3
      title: A cross-worktree id race is always caught and resolved by reallocate
      status: open
      tdd_phase: red
    - id: AC-4
      title: repolock's per-worktree lockfile scoping is confirmed intentional
      status: open
      tdd_phase: red
    - id: AC-5
      title: A sibling worktree's commit is confirmed unreachable from another's check
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — Generated verb sequences are checked for FSM legality each step

### AC-2 — Real subprocesses racing repolock never produce a duplicate id

### AC-3 — A cross-worktree id race is always caught and resolved by reallocate

### AC-4 — repolock's per-worktree lockfile scoping is confirmed intentional

### AC-5 — A sibling worktree's commit is confirmed unreachable from another's check

