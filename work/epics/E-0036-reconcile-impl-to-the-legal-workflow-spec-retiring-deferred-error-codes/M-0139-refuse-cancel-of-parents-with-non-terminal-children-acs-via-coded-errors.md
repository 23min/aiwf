---
id: M-0139
title: Refuse cancel of parents with non-terminal children/ACs via coded errors
status: in_progress
parent: E-0036
depends_on:
    - M-0138
tdd: required
acs:
    - id: AC-1
      title: Cancel of an epic with non-terminal child milestones refuses (coded)
      status: met
      tdd_phase: done
    - id: AC-2
      title: Cancel of a milestone with open ACs refuses (coded)
      status: met
      tdd_phase: done
    - id: AC-3
      title: Cancel still succeeds when all children/ACs are terminal
      status: open
      tdd_phase: green
    - id: AC-4
      title: Cancel codes retired from deferred and ac2KnownImplGaps lists
      status: open
      tdd_phase: red
---
## Goal

Add verb-time guards to `verb.Cancel` so `aiwf cancel E-NNNN` refuses (listing offenders) when any child milestone is non-terminal (D-0003), and `aiwf cancel M-NNNN` refuses when any AC is `open` (D-0004) — each carrying its structured code via M-0138's `CodedError` pattern.

## Context

Today both cancels succeed regardless of child state; M-0125's negative driver confirmed all four cells unguarded. The spec encodes the rule (`epic-cancel-non-terminal-children`, `milestone-cancel-non-terminal-acs`) but the impl is silent. The decisions chose **refuse-with-listing**, not auto-cascade — a parent with non-terminal children forces a per-child disposition decision before it terminalizes.

## Acceptance criteria

Each AC carries an explicit **Evidence** gate — the named test, driver cell, or drift policy that fails if the claim breaks. "Looks right" is not evidence.

### AC-1 — Cancel of an epic with non-terminal child milestones refuses (coded)

`aiwf cancel E-NNNN` with a non-terminal (`draft`/`in_progress`) child milestone refuses with the structured `epic-cancel-non-terminal-children` code, listing the offending milestone id(s), and leaves HEAD unchanged. The code is a `codes.Code{Class: codes.ClassLegality}` descriptor (D-0011), extractable via `entity.Code`. *Evidence:* the M-0125 negative driver's epic-cancel cell un-skipped (its `ac2KnownImplGaps` entries removed, an `errorSubstringsFor` mapping added) — binary-level assertion of non-zero exit + the listing + HEAD unchanged.

### AC-2 — Cancel of a milestone with open ACs refuses (coded)

`aiwf cancel M-NNNN` with an `open` AC refuses with the structured `milestone-cancel-non-terminal-acs` code, listing the offending composite id(s) `M-NNNN/AC-N`, and leaves HEAD unchanged. Reuses `entity.MilestoneCanGoDone` (the existing open-AC enumerator). The code is a `ClassLegality` descriptor. *Evidence:* the M-0125 negative driver's milestone-cancel cell un-skipped; same assertion shape.

### AC-3 — Cancel still succeeds when all children/ACs are terminal

`aiwf cancel` of an epic whose milestones are all terminal, or a milestone whose ACs are all terminal, still succeeds with the expected `cancelled` post-state. *Evidence:* the M-0124 positive driver cell for the legal cancel; binary-level success assertion — guards against a "just refuse everything" implementation.

### AC-4 — Cancel codes retired from deferred and ac2KnownImplGaps lists

The two codes (`epic-cancel-non-terminal-children`, `milestone-cancel-non-terminal-acs`) are removed from `deferredImplErrorCodes`, and the four cancel cells removed from `ac2KnownImplGaps`. *Evidence:* `TestM0123_AC5_SpecToImpl_ErrorCodesResolve` stays green after removal (the codes now resolve as real `ClassLegality` descriptors), and the M-0125 negative + M-0124 positive drivers stay green with the four cells live. (The closure that can't be claimed, only earned.)

## Constraints

- Codes emitted via M-0138's `CodedError` pattern, not `fmt.Errorf`.
- **Reviewed reconcile:** re-confirm D-0003 and D-0004 still hold before implementing.
- Refuse-with-listing only — no auto-cascade to children (per the decisions).
- `tdd: required`.

## Out of scope

The `CodedError` pattern itself (M-0138); classifier (M3), rename (M4), reachability (M5).

## Dependencies

M-0138. Closes G-0139.

