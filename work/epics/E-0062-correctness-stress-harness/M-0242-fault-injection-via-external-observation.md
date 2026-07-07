---
id: M-0242
title: Fault injection via external observation
status: draft
parent: E-0062
depends_on:
    - M-0240
tdd: required
acs:
    - id: AC-1
      title: A process killed while holding repolock releases it via kernel fd cleanup
      status: open
      tdd_phase: red
    - id: AC-2
      title: A process killed mid-write never leaves a half-written entity file
      status: open
      tdd_phase: red
    - id: AC-3
      title: Lock-held and temp-file states are detected with no production-code change
      status: open
      tdd_phase: red
    - id: AC-4
      title: A disk-full or permission-denied write surfaces as a clean error
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — A process killed while holding repolock releases it via kernel fd cleanup

### AC-2 — A process killed mid-write never leaves a half-written entity file

### AC-3 — Lock-held and temp-file states are detected with no production-code change

### AC-4 — A disk-full or permission-denied write surfaces as a clean error

