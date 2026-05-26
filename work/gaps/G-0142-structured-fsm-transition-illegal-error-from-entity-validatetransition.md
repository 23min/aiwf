---
id: G-0142
title: Structured fsm-transition-illegal error from entity.ValidateTransition
status: addressed
discovered_in: M-0123
addressed_by:
    - M-0138
---
## What's missing

`entity.ValidateTransition` returns free-form errors today:

```go
fmt.Errorf("%s status %q cannot transition to %q (allowed: %v)", k, from, to, allowed)
fmt.Errorf("%s status %q is terminal; cannot transition to %q", k, from, to)
```

The spec's `terminalIllegal` helper (`internal/workflows/spec/rules.go`)
references these refusals via the structured code `fsm-transition-illegal`,
used by every terminal-state cell across every kind plus AC and TDD-phase
sub-FSMs. The code is not emitted as a structured envelope today; consumers
have to string-match.

`fsm-transition-illegal` is listed in `deferredImplErrorCodes` (M-0123/AC-5)
with this gap as the tracking reason.

## Why it matters

CI scripts, downstream tools, and the AC-5 drift policy all want to
discriminate "this transition was refused by the FSM" from other verb
errors. Free-form errors force fragile string-matching. A structured code
is the kernel's standard surface for legality refusals; every other
finding code in `internal/check/hint.go` has one.

## Proposed fix shape

- Define a typed error in `internal/entity/` (e.g. `FSMTransitionError`
  with `Kind, From, To, Allowed` fields and a `Code()` method returning
  `"fsm-transition-illegal"`).
- `ValidateTransition` returns it instead of the bare `fmt.Errorf`.
- Verb callers (`promote`, `cancel`) catch the typed error and emit the
  code through their JSON envelope.
- Add a hint entry to `internal/check/hint.go`'s `hintTable` for
  `fsm-transition-illegal` (the policy
  `PolicyFindingCodesHaveHints` will fire otherwise).
- Once landed, remove `fsm-transition-illegal` from
  `deferredImplErrorCodes`.
