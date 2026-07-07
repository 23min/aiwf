---
id: M-0243
title: Named scenarios from G-0212 and G-0269
status: draft
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: A parallel-branch reallocate race is resolved per G-0212 item 1
      status: open
      tdd_phase: red
    - id: AC-2
      title: A concurrent cross-worktree edit-body race matches G-0212 item 2
      status: open
      tdd_phase: red
    - id: AC-3
      title: Archive-during-active-scope is exercised end-to-end per G-0212 item 3
      status: open
      tdd_phase: red
    - id: AC-4
      title: Force-push and cherry-pick vs acknowledge-illegal are exercised per G-0212
      status: open
      tdd_phase: red
    - id: AC-5
      title: G-0269's HEAD-drift race is scripted, expected-red until its guard lands
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — A parallel-branch reallocate race is resolved per G-0212 item 1

### AC-2 — A concurrent cross-worktree edit-body race matches G-0212 item 2

### AC-3 — Archive-during-active-scope is exercised end-to-end per G-0212 item 3

### AC-4 — Force-push and cherry-pick vs acknowledge-illegal are exercised per G-0212

### AC-5 — G-0269's HEAD-drift race is scripted, expected-red until its guard lands

