---
id: M-0141
title: Enforce three-edge scope reachability at verb-time
status: draft
parent: E-0036
depends_on:
    - M-0138
tdd: required
acs:
    - id: AC-1
      title: Scope reachability traverses exactly D-0006's three edges
      status: open
      tdd_phase: red
    - id: AC-2
      title: Out-of-scope authorized-agent verb refuses with structured code
      status: open
      tdd_phase: red
    - id: AC-3
      title: 'discovered_in reverse: agent promotes a gap it filed in scope'
      status: open
      tdd_phase: red
---
## Goal

Implement D-0006's three-edge scope reachability (target's `parent` chain reaches the scope-entity; composite-id containment; `discovered_in` reverse) and refuse, at verb-time, an authorized-agent verb whose target falls outside the open scope's reachable set — carrying a structured code via M-0138's pattern.

## Context

Greenfield: `internal/scope/scope.go` is FSM-only (open/pause/end, derived from commit-trailer history). No reachability check exists, so the kernel's "principal × agent × scope" commitment is partly hollow today — an authorized agent's verb is not actually constrained to its scope subtree. D-0006 already decides the model (three edges, governance edges explicitly excluded), so this milestone implements a decided thing — no new ADR.

## Acceptance criteria

- **AC1** — A reachability function resolves exactly the three edges from a scope-entity and excludes all governance edges (`depends_on`, `addressed_by`, `relates_to`, `supersedes`, `superseded_by`, `linked_adrs`). *Evidence:* table test over a fixture tree with one case per included edge (reachable) and one per excluded edge (not reachable) — every branch exercised.
- **AC2** — An authorized agent invoking a verb on a target inside the scope tree succeeds; on a target outside it, the verb refuses with the structured out-of-scope code. *Evidence:* binary-level positive + negative test (agent actor, open-scope fixture); assertion of exit + structured code + HEAD unchanged on the negative arm.
- **AC3** — The D-0006 friction case holds: an agent authorized on E-NN can promote a gap it filed with `--discovered-in M-K` where M-K is in E-NN's subtree. *Evidence:* dedicated test of the `discovered_in`-reverse arm (the case strict-parent-only reachability would wrongly refuse).
- **AC4** — The out-of-scope code is legality-classed (per M3) and referenced by a spec rule. *Evidence:* the AC-5 drift arm green with the new code present.

## Constraints

- Code emitted via M-0138's pattern.
- Edges exactly per D-0006 — closed set; a new kind needs explicit edge participation, never implicit graph traversal.
- **Reviewed reconcile:** read the impl against D-0006 and surface any divergence before coding.
- `tdd: required`.

## Out of scope

The `CodedError` foundation (M-0138); the other gaps.

## Design note

If at `aiwfx-start-milestone` this proves to need its own ADR or more than ~one milestone of impl (the verb-layer enforcement hook is the unsized risk), split it to its own epic per the epic spec's open question 1.

## Dependencies

M-0138 (and M3 for the classifier arm — soft). Closes G-0143.

### AC-1 — Scope reachability traverses exactly D-0006's three edges

### AC-2 — Out-of-scope authorized-agent verb refuses with structured code

### AC-3 — discovered_in reverse: agent promotes a gap it filed in scope

