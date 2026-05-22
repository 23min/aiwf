---
id: G-0163
title: ADR/Decision accepted cancel routes through FSM-illegal target
status: open
---
## Problem

`entity.CancelTarget(KindADR, StatusAccepted)` returns `StatusRejected`, and the same for `(KindDecision, StatusAccepted)`. The verb's `Cancel` path then applies the transition (accepted → rejected) **without an FSM legality check**. But the FSM forbids it: `entity.transitions[KindADR]["accepted"] = {"superseded"}` — `accepted → rejected` is NOT in the set.

Spec cell (Q3, `R-AUDIT-0014` / `R-FP-0021`):

```go
{
    Kind:              entity.KindADR,
    FromState:         "accepted",
    Verb:              "cancel",
    Outcome:           OutcomeIllegal,
    ExpectedErrorCode: "fsm-transition-illegal",
    RejectionLayer:    RejectionLayerVerbTime,
    BlockingStrict:    true,
},
```

M-0125's negative driver expects non-zero exit. Today the verb succeeds: `aiwf cancel ADR-0001 -> rejected`, commit lands, ADR is in `rejected` state despite the FSM forbidding the move.

The existing comment in `internal/entity/transition.go::CancelTarget` documents the design intent as *"status-agnostic; FSM legality of the move is upstream of CancelTarget's contract"* — but nobody upstream checks. This is a design seam that needs a chokepoint.

## Why it matters

Same shape as G-0162: the bad state lands in git history before any check fires. ADR/Decision `rejected` is meant to be reachable only from `proposed` (early-rejection path). Sneaking it from `accepted` violates the FSM's documented closure and the spec's Q3 explicit Illegal cell.

Symmetric for Decision: same FSM, same gap.

## Fix outline

Two options:

1. **State-aware CancelTarget** (mirror M-0131's Contract fix):

   ```go
   case KindADR, KindDecision:
       if currentStatus == StatusProposed {
           return StatusRejected
       }
       return ""  // accepted has no cancel target; FSM-illegal
   ```

   With this, the verb's `if target == ""` branch fires with `"(adr, accepted) has no cancel target"` — error contains "no cancel target", matches `errorSubstringsFor("fsm-transition-illegal")`.

   The existing `TestCancelTarget` cases for `adr-from-accepted` / `decision-from-accepted` need updating to expect `""` (matching new contract).

2. **Verb-side FSM check after CancelTarget**:

   ```go
   target := entity.CancelTarget(e.Kind, e.Status)
   if target != "" && !force {
       if err := entity.ValidateTransition(e.Kind, e.Status, target); err != nil {
           return nil, fmt.Errorf("cancel %s rejected: %w", id, err)
       }
   }
   ```

   Doesn't change CancelTarget's contract; adds the check at the verb boundary. Existing CancelTarget tests unaffected. Slightly more code.

Recommend (1) — it matches the M-0131 precedent (state-aware Contract) and aligns the function's actual returned value with the FSM's truth.

## Closing this gap

When the impl lands, remove `"adr-accepted-cancel"` from `ac2KnownImplGaps` (`internal/policies/m0125_negative_driver_test.go`) in the same commit. M-0125/AC-2 covers the cell automatically.

## Discovered in

M-0125/AC-2 driver dry-run.

## Status

`open`.
