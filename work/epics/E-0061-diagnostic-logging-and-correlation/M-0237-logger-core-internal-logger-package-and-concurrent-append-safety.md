---
id: M-0237
title: 'Logger core: internal/logger package and concurrent-append safety'
status: draft
parent: E-0061
tdd: required
acs:
    - id: AC-1
      title: Diagnostic logging defaults off, opt-in via env then aiwf.yaml
      status: open
      tdd_phase: red
    - id: AC-2
      title: Opted-in logs land in one daily XDG-state-home file, 30-day retention
      status: open
      tdd_phase: red
    - id: AC-3
      title: Concurrent writers to the shared log file never interleave or tear a line
      status: open
      tdd_phase: red
    - id: AC-4
      title: Bound logger fields never leak the operator's home-directory path
      status: open
      tdd_phase: red
    - id: AC-5
      title: atomic_write_chokepoint.go allowlists internal/logger's append write
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — Diagnostic logging defaults off, opt-in via env then aiwf.yaml

### AC-2 — Opted-in logs land in one daily XDG-state-home file, 30-day retention

### AC-3 — Concurrent writers to the shared log file never interleave or tear a line

### AC-4 — Bound logger fields never leak the operator's home-directory path

### AC-5 — atomic_write_chokepoint.go allowlists internal/logger's append write

