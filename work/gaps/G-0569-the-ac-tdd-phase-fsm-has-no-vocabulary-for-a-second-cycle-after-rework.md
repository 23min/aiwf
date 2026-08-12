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

**Two routes, measured 2026-08-08.**

*Reset the phase on arrival at `open`.* The demote stays sovereign; only the
carried-over phase changes. It needs no FSM edit, no policy change and no
amendment to a normative rule, and it makes the repair path that already exists
honest rather than adding a new one. Keying the reset on arrival rather than on
the `met → open` edge also covers the `deferred → open` door, where a forced
demote carries `done` across in exactly the same way. It interacts with the
verb's same-state guard, which compares status alone: under a target-keyed reset
the guard's claim spans two fields and must compare both, or it converges while
a stale phase survives.

*Add `met → open` as an ordinary edge.* Measured cost is larger than it looks.
It fails the FSM-invariants policy as a cycle, which reaches an accepted ADR
stating the FSM is one-directional, plus the numbered legal-workflows rule that
enumerates `met`'s legal targets, plus a spec cell the drift tests structurally
cannot demand — they assert every *from-state* has a rule, so `met` already
having rules means a new target is invisible. The edge is also actively unsafe
without the phase reset landing first: because `done` is terminal in the phase
FSM, a reopened AC cannot re-enter its cycle, so the second `met` rides on the
first cycle's evidence with no force and no trailer anywhere.

A decision taking this route has to answer three things together: the phase
semantics on reopen, whether reopening completed work should be untrailered at
all, and the still-open question of `deferred → open` — the easier half of the
same question, which the first-principles rules leave explicitly unresolved.

Related: the test-metrics rule is satisfied for an AC's whole life by any single
cycle's trailer, so the phase reset restores one guard while that one stays soft.

