---
id: M-0258
title: Race concurrent promote/cancel/AC operations against shared entity state
status: draft
parent: E-0065
depends_on:
    - M-0257
tdd: required
acs:
    - id: AC-1
      title: N concurrent actors race promote/cancel on one shared milestone+AC
      status: open
      tdd_phase: red
    - id: AC-2
      title: Oracle distinguishes a legitimate race from a guard violation
      status: open
      tdd_phase: red
    - id: AC-3
      title: Re-running against a reintroduced G-0335-shaped regression fails the run
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — N concurrent actors race promote/cancel on one shared milestone+AC

### AC-2 — Oracle distinguishes a legitimate race from a guard violation

### AC-3 — Re-running against a reintroduced G-0335-shaped regression fails the run

