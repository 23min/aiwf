---
id: M-075
title: Lifecycle-gate entity-body-empty rule (closes G-071)
status: draft
parent: E-22
tdd: required
acs:
    - id: AC-1
      title: entity.IsTerminal(kind, status) helper available
      status: open
      tdd_phase: red
    - id: AC-2
      title: Rule skips terminal-status entities
      status: open
      tdd_phase: red
    - id: AC-3
      title: Rule skips ACs whose parent milestone is draft
      status: open
      tdd_phase: red
    - id: AC-4
      title: Rule still fires on active-state entities with empty sections
      status: open
      tdd_phase: red
    - id: AC-5
      title: Warning baseline on kernel tree drops by 27
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — entity.IsTerminal(kind, status) helper available

### AC-2 — Rule skips terminal-status entities

### AC-3 — Rule skips ACs whose parent milestone is draft

### AC-4 — Rule still fires on active-state entities with empty sections

### AC-5 — Warning baseline on kernel tree drops by 27

