---
id: G-0569
title: The AC tdd_phase FSM has no vocabulary for a second cycle after rework
status: open
priority: medium
discovered_in: M-0300
---
## What's missing

An AC's `tdd_phase` FSM is linear and terminal: `"" → red → green → (refactor →)
done`, with `done` having no outgoing edges. The AC status FSM is nearly as
tight — `met` reaches only `deferred` and `cancelled`, never back to `open`.

Together these model an AC that is worked once. They have no vocabulary for an
AC that was promoted to `met`, had its evidence refuted, and is being worked
again. Both reverts require `--force --reason`, which is the sovereign
exceptional path, and the phase field then has to be forced backwards along an
FSM whose linearity exists specifically to prevent a "green without red" claim.

## Why it matters

Rework after a refuted claim is not exceptional in the sense `--force` is for.
It is what an independent review is supposed to cause, and this repo runs one at
every milestone wrap.

The mechanical consequence is worse than the bookkeeping one. `acs-tdd-audit`
enforces "met requires `tdd_phase: done`". An AC reverted to `open` keeps
`done`, so the audit is satisfied in advance for the second cycle: the AC can be
re-promoted to `met` without a failing test ever having been written, and no
check objects. The guard goes soft exactly when a review has just demonstrated
the evidence bar was too low.

Discovered while reverting M-0300's three ACs after review refuted them; the
phases were forced back to `red` one `--force` at a time to re-arm the audit.
