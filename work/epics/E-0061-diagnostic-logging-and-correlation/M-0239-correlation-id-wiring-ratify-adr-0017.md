---
id: M-0239
title: Correlation id wiring; ratify ADR-0017
status: draft
parent: E-0061
depends_on:
    - M-0237
    - M-0238
tdd: required
acs:
    - id: AC-1
      title: An envelope's correlation_id matches the run_id in that invocation's log lines
      status: open
      tdd_phase: red
    - id: AC-2
      title: Mutating verbs report per-verb-appropriate metadata in their envelope
      status: open
      tdd_phase: red
    - id: AC-3
      title: An operator can pass --trace to see per-phase timings via the logger
      status: open
      tdd_phase: red
    - id: AC-4
      title: A renamed Envelope field is caught by a structural policy test
      status: open
      tdd_phase: red
    - id: AC-5
      title: ADR-0017 reads accepted with CLAUDE.md matching shipped behavior
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — An envelope's correlation_id matches the run_id in that invocation's log lines

### AC-2 — Mutating verbs report per-verb-appropriate metadata in their envelope

### AC-3 — An operator can pass --trace to see per-phase timings via the logger

### AC-4 — A renamed Envelope field is caught by a structural policy test

### AC-5 — ADR-0017 reads accepted with CLAUDE.md matching shipped behavior

