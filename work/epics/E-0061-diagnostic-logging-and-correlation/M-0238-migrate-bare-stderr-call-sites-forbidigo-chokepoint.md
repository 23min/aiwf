---
id: M-0238
title: Migrate bare-stderr call sites; forbidigo chokepoint
status: draft
parent: E-0061
depends_on:
    - M-0237
tdd: required
acs:
    - id: AC-1
      title: Named bare-stderr call sites emit diagnostic events through the bound logger
      status: open
      tdd_phase: red
    - id: AC-2
      title: A migrated verb run with AIWF_LOG=info fires the expected structured event
      status: open
      tdd_phase: red
    - id: AC-3
      title: A non-allowlisted bare print call fails CI via forbidigo and a policy test
      status: open
      tdd_phase: red
    - id: AC-4
      title: aiwf.yaml's logging block is parsed, validated, and surfaced by aiwf doctor
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — Named bare-stderr call sites emit diagnostic events through the bound logger

### AC-2 — A migrated verb run with AIWF_LOG=info fires the expected structured event

### AC-3 — A non-allowlisted bare print call fails CI via forbidigo and a policy test

### AC-4 — aiwf.yaml's logging block is parsed, validated, and surfaced by aiwf doctor

