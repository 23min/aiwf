---
id: M-072
title: aiwf list verb, status filter-helper refactor, contract-skill drift fix
status: draft
parent: E-20
tdd: required
acs:
    - id: AC-1
      title: Core flag set works end-to-end
      status: open
      tdd_phase: red
    - id: AC-2
      title: 'JSON envelope: result is array of summary objects'
      status: open
      tdd_phase: red
    - id: AC-3
      title: Default excludes terminal status; --archived includes them
      status: open
      tdd_phase: red
    - id: AC-4
      title: entity.IsTerminal(kind, status) helper added
      status: open
      tdd_phase: red
    - id: AC-5
      title: Closed-set completion wired for --kind and --status
      status: open
      tdd_phase: red
    - id: AC-6
      title: Shared filter helper extracted; status uses it
      status: open
      tdd_phase: red
    - id: AC-7
      title: Status text and JSON goldens unchanged after refactor
      status: open
      tdd_phase: red
    - id: AC-8
      title: contracts-plan and contract-skill drift fixed
      status: open
      tdd_phase: red
    - id: AC-9
      title: Verb-level integration test drives the dispatcher
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — Core flag set works end-to-end

### AC-2 — JSON envelope: result is array of summary objects

### AC-3 — Default excludes terminal status; --archived includes them

### AC-4 — entity.IsTerminal(kind, status) helper added

### AC-5 — Closed-set completion wired for --kind and --status

### AC-6 — Shared filter helper extracted; status uses it

### AC-7 — Status text and JSON goldens unchanged after refactor

### AC-8 — contracts-plan and contract-skill drift fixed

### AC-9 — Verb-level integration test drives the dispatcher

