---
id: G-0506
title: PromoteACPhase refuses from working-copy bytes no verb wrote
status: open
discovered_in: M-0285
---
## What's missing

`PromoteACPhase` reads `ac.TDDPhase` off the loaded tree, consults the phase FSM,
and refuses by naming that value — with no claim-side guard. Its three siblings in
the same file (`promoteAC`, `cancelAC`, `renameAC`) all call `guardClaim` in their
preludes.

Measured, on a disposable repo with a milestone at `tdd: required`:

    committed:  tdd_phase: red
    hand-edit:  tdd_phase: done
    aiwf promote M-NNNN/AC-1 --phase green
    -> AC tdd_phase "done" cannot transition to "green"

HEAD says `red`, where `red -> green` is legal. The operator is told the FSM forbids
their transition and sent to repair a record that is intact.

## Why it matters

The refusal arm produces no file ops, so the commit-side guard in `verb.Apply` never
runs — there is no plan for it to inspect. This is the window ADR-0038 opened the
claim-side seam to close, and the ADR names the same shape in `promote`'s resolver
re-point refusal: a verdict computed from bytes HEAD contradicts, reported as fact
about the record.

It is not caught by `PolicyClaimGuardPresence`. That policy derives its subject from
`noOpClaimScopes`, whose key set is "constructs `Result{NoOp: true}`".
`PromoteACPhase` does not converge, so it has no row and no guard requirement. The
two sets differ — "decides from loaded state before producing ops" is wider than
"converges" — and the ledger has no way to say so.

## Scope

Route `PromoteACPhase` through `guardClaim` on the parent milestone's path, as its
three siblings do, with a same-value-over-divergent test.

Then decide what records it. A guarded non-converging route cannot take a
`noOpClaimScopes` row today: that ledger reports "recorded but no longer converges"
for any row naming a non-converging function, so adding one fails. Either widen the
ledger's key set to the routes that decide from loaded state, or give the guard
requirement its own ledger keyed on that property.

## References

- ADR-0038 — the two guard seams, and the refusal-window shape this instance repeats
- M-0285 — the invariant whose converging-routes-only bound this sits outside
