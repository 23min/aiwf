---
id: M-0137
title: 'fsm-history-consistent: batched git ops + silent-swallow fix'
status: draft
parent: E-0033
depends_on:
    - M-0130
tdd: required
acs:
    - id: AC-1
      title: internal/gitops/ bulk-revwalk helper streams (commit, parent, paths, trailers)
      status: open
      tdd_phase: red
    - id: AC-2
      title: internal/gitops/ cat-file --batch content-reader pump
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'fsm-history-consistent: no per-entity exec.Command — routes through helpers'
      status: open
      tdd_phase: red
    - id: AC-4
      title: history-walk-error subcode emits per failed entity (severity error)
      status: open
      tdd_phase: red
    - id: AC-5
      title: Walker continues past per-entity errors; partial findings preserved
      status: open
      tdd_phase: red
    - id: AC-6
      title: 'Negative test: per-entity walk failure surfaces history-walk-error'
      status: open
      tdd_phase: red
    - id: AC-7
      title: 'Perf regression test: kernel tree aiwf check completes within baseline budget'
      status: open
      tdd_phase: red
    - id: AC-8
      title: Audit catalog R-RULE-149 updated to list all four subcodes with severities
      status: open
      tdd_phase: red
    - id: AC-9
      title: 'G-0148 body updated: fsm-history slice closed; perf retrofits remain open'
      status: open
      tdd_phase: red
---

## Goal

## Acceptance criteria

### AC-1 — internal/gitops/ bulk-revwalk helper streams (commit, parent, paths, trailers)

### AC-2 — internal/gitops/ cat-file --batch content-reader pump

### AC-3 — fsm-history-consistent: no per-entity exec.Command — routes through helpers

### AC-4 — history-walk-error subcode emits per failed entity (severity error)

### AC-5 — Walker continues past per-entity errors; partial findings preserved

### AC-6 — Negative test: per-entity walk failure surfaces history-walk-error

### AC-7 — Perf regression test: kernel tree aiwf check completes within baseline budget

### AC-8 — Audit catalog R-RULE-149 updated to list all four subcodes with severities

### AC-9 — G-0148 body updated: fsm-history slice closed; perf retrofits remain open

