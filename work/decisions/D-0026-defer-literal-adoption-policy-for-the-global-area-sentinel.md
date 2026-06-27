---
id: D-0026
title: Defer literal-adoption policy for the global area sentinel
status: proposed
---
## Context

M-0184 introduces `global` as a reserved value of the single-valued `area` dimension
(ADR-0021), defined once as `entity.AreaGlobal` and routed through the SSOT predicate
`entity.IsValidAreaValue`. The repo's policy culture (`enum_literal_adoption`,
`closed_set_status_constants` under `internal/policies/`) would normally add a chokepoint
forbidding a bare `== "global"` comparison outside the token's home package, so the predicate
stays the single definition of "valid area value".

## Decision

Do not add a literal-adoption policy for the `global` area token now. AC-1's behavioral unit
test pins the predicate's classification (global / declared member / unknown); the four consumer
sites (`area-unknown`, `set-area`, `add --area`, the read-filter note) are confirmed to route
through the predicate by their own AC tests and by review — there is no parallel `== "global"`
check today.

Rationale:

- One reserved token, one definition site (`entity.AreaGlobal`), and a handful of consumers —
  a mechanical chokepoint for a single literal is premature (YAGNI; abstract on the third).
- The status closed-set policies key on clean syntactic markers — a `Status:` assignment, a
  `.Status` comparison, a trailer `Value:`, a `switch` `case` label. An area-value comparison
  has no such marker, so a scanner for bare `"global"` literals would be noisy and
  false-positive-prone — exactly the "too noisy to mechanize" class the repo keeps advisory.

## Revisit trigger

Add the literal-adoption policy when a **third** reserved area value is introduced. At three,
the reserved set is large enough that a parallel `== "..."` check becomes a real drift risk
worth a chokepoint; until then the single behavioral test plus review is the proportionate
guard.

## References

- ADR-0021 — the `global` reserved-value decision this scopes the enforcement of.
- M-0184/AC-1 — the SSOT predicate (`entity.IsValidAreaValue`) whose enforcement depth this
  decision sets.
- `internal/policies/enum_literal_adoption.go`, `internal/policies/closed_set_status_constants.go`
  — the closed-set literal-adoption precedent this deliberately diverges from.
