---
id: M-070
title: aiwf doctor warning for missing recommended plugins
status: draft
parent: E-18
tdd: required
acs:
    - id: AC-1
      title: doctor.recommended_plugins config field accepts list
      status: open
      tdd_phase: red
    - id: AC-2
      title: doctor reads installed_plugins.json and matches project scope
      status: open
      tdd_phase: red
    - id: AC-3
      title: Each missing plugin emits one warning with install command
      status: open
      tdd_phase: red
    - id: AC-4
      title: Empty config list means no checks fire (kernel-neutral)
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — doctor.recommended_plugins config field accepts list

### AC-2 — doctor reads installed_plugins.json and matches project scope

### AC-3 — Each missing plugin emits one warning with install command

### AC-4 — Empty config list means no checks fire (kernel-neutral)

