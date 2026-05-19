---
id: G-0139
title: Implement cancel-cascade per D-0003 and D-0004
status: open
discovered_in: M-0123
---
## What's missing

Two preconditioned-illegal cells in `internal/workflows/spec/rules.go` reference
verb-time finding codes that the impl does not yet emit:

- `epic-cancel-non-terminal-children` — fires when `aiwf cancel E-NNNN` is
  invoked while any child milestone is still non-terminal (`draft` or
  `in_progress`). Per **D-0003** (committed in M-0123 phase 1) the verb refuses
  with a listing of the non-terminal children.

- `milestone-cancel-non-terminal-acs` — fires when `aiwf cancel M-NNNN` is
  invoked while any AC is still in a non-terminal status (`open`). Per **D-0004**
  the verb refuses with a listing of the non-terminal ACs.

Today both verbs cancel without consulting the child state. The kernel's status
FSM allows the transition; the spec says it should be guarded.

Both codes are listed in `internal/policies/m0123_ac5_drift_test.go`'s
`deferredImplErrorCodes` allowlist with this gap as the tracking reason. When
the impl lands, the allowlist entries come out and the M-0123/AC-5 drift test
re-binds the spec cells to the impl-side `Code: "..."` literals.

## Why it matters

Without the guards, an operator can cancel an epic mid-flight and orphan
its in-flight milestones — same for a milestone with open ACs. The spec
already encodes the rule; the impl needs to catch up so the spec stops being
aspirational.

## Proposed fix shape

- Add the listing logic at the `aiwf cancel` verb's entrypoint
  (`internal/cli/cancel/`). On a refused cancel, return exit-code 2 with the
  structured error code and a multi-line message listing the non-terminal
  children/ACs.
- Test: kernel-level fixture trees with one non-terminal child (epic) and
  one open AC (milestone); assert the verb refuses with the expected code.
- Once landed, remove the two entries from `deferredImplErrorCodes` in
  `internal/policies/m0123_ac5_drift_test.go`.
