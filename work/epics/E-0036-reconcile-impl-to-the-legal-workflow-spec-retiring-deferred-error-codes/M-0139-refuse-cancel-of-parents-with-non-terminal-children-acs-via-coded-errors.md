---
id: M-0139
title: Refuse cancel of parents with non-terminal children/ACs via coded errors
status: draft
parent: E-0036
depends_on:
    - M-0138
tdd: required
---
## Goal

Add verb-time guards to `verb.Cancel` so `aiwf cancel E-NNNN` refuses (listing offenders) when any child milestone is non-terminal (D-0003), and `aiwf cancel M-NNNN` refuses when any AC is `open` (D-0004) — each carrying its structured code via M-0138's `CodedError` pattern.

## Context

Today both cancels succeed regardless of child state; M-0125's negative driver confirmed all four cells unguarded. The spec encodes the rule (`epic-cancel-non-terminal-children`, `milestone-cancel-non-terminal-acs`) but the impl is silent. The decisions chose **refuse-with-listing**, not auto-cascade — a parent with non-terminal children forces a per-child disposition decision before it terminalizes.

## Acceptance criteria

- **AC1** — `aiwf cancel E-NNNN` with a non-terminal (`draft`/`in_progress`) child milestone refuses with structured `epic-cancel-non-terminal-children`, listing the offending milestone id(s). *Evidence:* M-0125 negative cell (epic-cancel) un-skipped; binary-level assertion of non-zero exit + structured code + HEAD unchanged + the listing.
- **AC2** — `aiwf cancel M-NNNN` with an `open` AC refuses with structured `milestone-cancel-non-terminal-acs`, listing the offending composite id(s). *Evidence:* M-0125 negative cell (milestone-cancel) un-skipped; same assertion shape.
- **AC3** — `aiwf cancel` of an epic/milestone whose children are all terminal still succeeds with the expected post-state. *Evidence:* the M-0124 positive driver cell for the legal cancel; binary-level success assertion (guards against "just refuse everything").
- **AC4** — The two codes are removed from `deferredImplErrorCodes`; the four cancel cells removed from `ac2KnownImplGaps`. *Evidence:* `TestM0123_AC5_SpecToImpl_ErrorCodesResolve` green after removal; M-0125 driver green with the four cells live.

## Constraints

- Codes emitted via M-0138's `CodedError` pattern, not `fmt.Errorf`.
- **Reviewed reconcile:** re-confirm D-0003 and D-0004 still hold before implementing.
- Refuse-with-listing only — no auto-cascade to children (per the decisions).
- `tdd: required`.

## Out of scope

The `CodedError` pattern itself (M-0138); classifier (M3), rename (M4), reachability (M5).

## Dependencies

M-0138. Closes G-0139.
