---
id: M-0054
title: Dynamic id completion and drift test
status: done
parent: E-0014
acs:
    - id: AC-1
      title: --epic=<TAB> enumerates live epic ids from the planning tree
      status: met
    - id: AC-2
      title: Graceful no-op completion when cwd is not a valid aiwf project
      status: met
    - id: AC-3
      title: Drift-prevention policy test fails CI on flag without completion wiring
      status: met
    - id: AC-4
      title: Test covers static-completion and dynamic-completion required cases
      status: met
---

## Goal

Wire dynamic completion of live entity ids — `--epic=<TAB>`, `--milestone=<TAB>`, etc. enumerate from the current planning tree. Add an `internal/policies/` test that fails CI when a flag has neither a completion annotation nor an explicit opt-out — the mechanical chokepoint that backs the auto-completion design principle.

## Approach

`ValidArgsFunction` shells back to aiwf to enumerate ids. Graceful degradation when cwd isn't a project: return empty completions, no error spam — the user just sees no suggestions, not a crash. The drift-prevention policy test enumerates Cobra's flag tree and asserts every flag has either a completion function bound or a documented opt-out, in the spirit of the existing `internal/policies/` tests.

## Acceptance criteria

### AC-1 — --epic=<TAB> enumerates live epic ids from the planning tree

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-054/AC-1` for the actual implementation history._

### AC-2 — Graceful no-op completion when cwd is not a valid aiwf project

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-054/AC-2` for the actual implementation history._

### AC-3 — Drift-prevention policy test fails CI on flag without completion wiring

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-054/AC-3` for the actual implementation history._

### AC-4 — Test covers static-completion and dynamic-completion required cases

_Grandfathered: this AC was met before M-0066's `entity-body-empty` rule made body prose required. The behavior pinned here is recorded in the AC's commit chain — see `aiwf history M-054/AC-4` for the actual implementation history._
