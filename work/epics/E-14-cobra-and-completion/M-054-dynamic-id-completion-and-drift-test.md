---
id: M-054
title: Dynamic id completion and drift test
status: draft
parent: E-14
---

## Goal

Wire dynamic completion of live entity ids — `--epic=<TAB>`, `--milestone=<TAB>`, etc. enumerate from the current planning tree. Add an `internal/policies/` test that fails CI when a flag has neither a completion annotation nor an explicit opt-out — the mechanical chokepoint that backs the auto-completion design principle.

## Approach

`ValidArgsFunction` shells back to aiwf to enumerate ids. Graceful degradation when cwd isn't a project: return empty completions, no error spam — the user just sees no suggestions, not a crash. The drift-prevention policy test enumerates Cobra's flag tree and asserts every flag has either a completion function bound or a documented opt-out, in the spirit of the existing `internal/policies/` tests.

## Acceptance criteria
