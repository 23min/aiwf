---
id: M-054
title: Dynamic id completion and drift test
status: draft
parent: E-14
acs:
    - id: AC-1
      title: --epic=<TAB> enumerates live epic ids from the planning tree
      status: open
    - id: AC-2
      title: Graceful no-op completion when cwd is not a valid aiwf project
      status: open
    - id: AC-3
      title: Drift-prevention policy test fails CI on flag without completion wiring
      status: open
---

## Goal

Wire dynamic completion of live entity ids — `--epic=<TAB>`, `--milestone=<TAB>`, etc. enumerate from the current planning tree. Add an `internal/policies/` test that fails CI when a flag has neither a completion annotation nor an explicit opt-out — the mechanical chokepoint that backs the auto-completion design principle.

## Approach

`ValidArgsFunction` shells back to aiwf to enumerate ids. Graceful degradation when cwd isn't a project: return empty completions, no error spam — the user just sees no suggestions, not a crash. The drift-prevention policy test enumerates Cobra's flag tree and asserts every flag has either a completion function bound or a documented opt-out, in the spirit of the existing `internal/policies/` tests.

## Acceptance criteria

### AC-1 — --epic=<TAB> enumerates live epic ids from the planning tree

### AC-2 — Graceful no-op completion when cwd is not a valid aiwf project

### AC-3 — Drift-prevention policy test fails CI on flag without completion wiring

