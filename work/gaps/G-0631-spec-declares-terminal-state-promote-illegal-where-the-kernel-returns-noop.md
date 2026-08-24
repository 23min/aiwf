---
id: G-0631
title: Spec declares terminal-state promote illegal where the kernel returns NoOp
status: open
priority: high
---
## What's missing

The legal-workflow spec table declares every terminal-state `promote` coordinate
`OutcomeIllegal` with `fsm-transition-illegal`. M-0281/AC-1 — `met`, on a `done`
milestone — makes a promote to the entity's current status a NoOp;
`internal/verb/promote.go:96` returns exit 0 with *"is already X; nothing to
change"*. So the spec asserts a refusal at a coordinate where the kernel succeeds.

The cause is the cell key. `Rule` carries `Kind`, `FromState` and `Verb` and no
target, so one cell covers both `promote` to `cancelled` from a `done` epic —
genuinely illegal — and `promote` to `done` from a `done` epic, which is the NoOp.
The single declared outcome is right for the first and wrong for the second.

Measured 2026-08-24 on `main`, enumerating every (Kind, FromState, Verb)
coordinate against `spec.Rules()`:

- 99 coordinates over the three legality verbs; 49 declared, 50 undeclared.
- All 15 terminal-state `promote` coordinates are declared, all
  `OutcomeIllegal` / `fsm-transition-illegal`.
- `promote` is otherwise complete: 33 of 33 coordinates declared.

The negative-cell driver already compensates for the divergence rather than
reporting it. `internal/policies/m0125_negative_driver_test.go` steers its cases
away from the same-status path, noting in a comment that same-status promote is a
NoOp per M-0281 and no longer a rejection. The suite sided with the kernel; the
table was never corrected.

The bidirectional drift policy does not catch this. Its arms compare coordinates,
kinds, states and finding codes between `spec.Rules()` and the implementation.
Nothing compares a declared outcome against what the verb actually returns at that
coordinate.

## Why it matters

An accepted milestone changed kernel behavior, and the artifact whose whole purpose
is to encode kernel behavior kept the superseded answer. A reader of the spec — or
of any reference rendered from it — is told a refusal happens where the kernel
returns success.

This is the reason the spec cannot yet serve as the never-drifting legality
reference it was built to be: the drift policy guards its shape and its
vocabulary, not its verdicts.

Resolution shape: the target belongs in the cell key, so that a NoOp is expressible
as `FromState` equal to the target and an illegal transition is expressible
separately at the same origin. That is a schema change to `Rule`, not a correction
to these 15 cells, and it carries the rest of the table with it.
