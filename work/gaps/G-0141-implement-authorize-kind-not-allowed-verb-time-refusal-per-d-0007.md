---
id: G-0141
title: Implement authorize-kind-not-allowed verb-time refusal per D-0007
status: open
discovered_in: M-0123
---
## What's missing

Per **D-0007** (committed in M-0123 phase 1), `aiwf authorize` should refuse
when the scope-entity is not a `KindEpic` or `KindMilestone`. The other four
kinds (gap, decision, contract, ADR) are not delegation targets — they have
no "in-flight" state for an agent to advance.

The spec's `authorizeKindRestrictionRules()` in
`internal/workflows/spec/rules.go` encodes four illegal cells (one per
disallowed kind) with `ExpectedErrorCode: "authorize-kind-not-allowed"`.
The impl today accepts authorize for any kind.

`authorize-kind-not-allowed` is listed in `deferredImplErrorCodes`
(M-0123/AC-5) with this gap as the tracking reason.

## Why it matters

`aiwf authorize G-NNNN --to ai/claude` succeeds today, opening a scope on
a gap entity that has no FSM-driven work surface. The agent has nothing to
advance; the scope sits there until manually paused. A verb-time refusal
saves the operator from the silent-success failure mode.

## Proposed fix shape

- In `internal/cli/authorize/` (or wherever the authorize verb's entry
  point lives), look up the target entity, read its `Kind`, and refuse
  with `authorize-kind-not-allowed` if the kind is not in
  `{KindEpic, KindMilestone}`.
- Exit code 2; multi-line message names the four disallowed kinds.
- Test: fixture trees for each of the four disallowed kinds; verb
  refuses with the expected code for each.
- Once landed, remove `authorize-kind-not-allowed` from
  `deferredImplErrorCodes`.
