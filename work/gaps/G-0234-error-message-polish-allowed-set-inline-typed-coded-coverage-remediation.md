---
id: G-0234
title: 'Error-message polish: allowed-set inline, typed Coded coverage, remediation'
status: open
---
## What's missing

Four small error-handling consistency gaps surfaced under E4 and E2:

1. **Internal-symbol references in error messages.** A few errors point at internal Go symbols (`"see acTransitions"`, `"see tddPhaseTransitions"`) — invisible to operators reading the error. Replace with the allowed-set inline ("expected one of: draft, active, paused, done"). One-time pass; the typed `FSMTransitionError` already lists the set internally, so this is a serialization shape change, not a data change.
2. **Extend typed `Coded`-error pattern.** A subset of verb errors implement `entity.Coded`; the rest are bare-wrapped `fmt.Errorf`. The `Coded` shape carries the machine-readable code that the check-rule layer also uses — extending it everywhere gives downstream tooling a uniform error vocabulary. One pass through `internal/verb/*.go` error sites; rough scope ~30 sites.
3. **Remediation sentences in short flag-validation errors.** A few flag-validation errors are one-line ("missing required flag --foo") without remediation. Add the corrective ("missing required flag --foo; pass --foo=<value> to set it"). Sized as a one-PR sweep over `cmd/aiwf/` and `internal/cli/<verb>/`.
4. **Name disk-full / ENOSPC explicitly** in `internal/verb/apply.go`'s failure-mode doc. Today the path is covered by the generic write-wrap that triggers rollback, but the docblock doesn't enumerate it; an operator reading the comment for "what does Apply do under disk-full" gets the answer only by reading the implementation.

## Why it matters

E4's Strong verdict noted that aiwf's error discipline is "unusually disciplined" — the polishes above are the long-tail of consistency. None are bugs; each makes the error surface uniformly self-explaining.

## Source

`docs/pocv3/health-scorecard-2026-06-04.md` §E4 (all three moves), §E2 (move 2: ENOSPC naming).
