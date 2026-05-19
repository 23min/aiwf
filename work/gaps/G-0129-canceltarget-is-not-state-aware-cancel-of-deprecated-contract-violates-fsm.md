---
id: G-0129
title: 'CancelTarget is not state-aware: cancel of deprecated Contract violates FSM'
status: open
discovered_in: M-0121
---
## What's missing

`internal/entity/transition.go`'s `CancelTarget(kind)` returns a single terminal-cancel status per kind, regardless of the entity's current state. For five of the six kinds this is correct — they have single-line lifecycles where the same terminal cancel applies. For **Contract**, it's wrong:

```go
case KindContract:
    return "rejected"
```

But the Contract FSM is:

```
proposed → accepted → deprecated → retired
       ↘     ↘          (no edge to "rejected")
          rejected
```

There is **no FSM edge from `deprecated` to `rejected`**. So when an operator runs `aiwf cancel C-NNN` against a deprecated contract, `CancelTarget` returns `"rejected"`, the verb calls `ValidateTransition(KindContract, "deprecated", "rejected")`, the validator errors with *"contract status `deprecated` cannot transition to `rejected`"*, and the verb fails. The user is stuck — they can't cancel a deprecated contract, even though that's a legitimate lifecycle move (the natural terminal from `deprecated` is `retired`, not `rejected`).

## Why it matters

This was surfaced as **review finding #1** during M-0121's audit catalog work (2026-05-18). It's a real correctness bug, not a doc issue: `aiwf cancel` is documented as the canonical terminal-state move, but for one specific (kind, state) combination it fails for a structural reason.

The audit catalog's R-RULE-021 was revised to describe the *correct* semantic (state-aware target). The code does not yet match.

## Two viable fixes

**(a) Make `CancelTarget` state-aware.** Refactor `CancelTarget(kind) string` to `CancelTarget(kind, currentStatus) string`:

```go
case KindContract:
    switch currentStatus {
    case "proposed", "accepted":
        return "rejected"
    case "deprecated":
        return "retired"
    }
```

For other kinds the new parameter is unused; the function collapses to today's behavior. The verb's call site changes from `CancelTarget(kind)` to `CancelTarget(kind, entity.Status)`.

**(b) Refuse `aiwf cancel` on `deprecated` Contracts; operator uses `aiwf promote retired`.** Smaller code change (just an early-return with a helpful error), but worse UX — `aiwf cancel` should "just work" as the terminal-move verb.

**Lean: (a).** It's the principled fix. The function-signature change is localized to the cancel verb's dispatch. Tests are straightforward (a positive case for each kind + state combination).

## Implementation cost

Half a milestone or a single `wf-patch`:
- Update `CancelTarget` signature + per-kind switch logic
- Update the cancel verb's call site
- Add positive tests for each (kind, current-state) → target mapping
- Add a negative test: cancel from a state with no legal terminal target errors (the `deprecated → ?` case for kinds that don't define one)
- Update audit catalog's R-AUDIT-0032 + R-RULE-021 to remove the "code bug" qualifier

## Related

- Audit catalog R-RULE-021, R-AUDIT-0032 reference this gap
- `internal/entity/transition.go::CancelTarget`
- Review finding #1 in M-0121's external review (2026-05-18)
