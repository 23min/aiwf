---
id: M-0240
title: 'Harness skeleton: driver, scenario interface, streaming report'
status: draft
parent: E-0062
depends_on:
    - M-0239
tdd: required
acs:
    - id: AC-1
      title: A stress run builds the aiwf binary under test once, never trusting PATH
      status: open
      tdd_phase: red
    - id: AC-2
      title: Each raw-report event appends via a single Write call
      status: open
      tdd_phase: red
    - id: AC-3
      title: A run killed mid-scenario still composes without failing on a truncated line
      status: open
      tdd_phase: red
    - id: AC-4
      title: A scenario's repo is cleaned up on pass and preserved on fail
      status: open
      tdd_phase: red
    - id: AC-5
      title: A --repeat N flag reruns a scenario N times with a logged seed per attempt
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — A stress run builds the aiwf binary under test once, never trusting PATH

### AC-2 — Each raw-report event appends via a single Write call

### AC-3 — A run killed mid-scenario still composes without failing on a truncated line

### AC-4 — A scenario's repo is cleaned up on pass and preserved on fail

### AC-5 — A --repeat N flag reruns a scenario N times with a logged seed per attempt

