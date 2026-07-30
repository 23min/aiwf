---
id: G-0458
title: 'AC tdd_phase same-phase input: converge to NoOp or keep refusing'
status: open
priority: medium
discovered_in: M-0281
---
## What's missing

`aiwf promote <M>/AC-<N> --phase <p>` is the one mutating verb left outside the
same-state NoOp convention. When the requested phase equals the AC's current
`tdd_phase`, the TDD-phase FSM (`"" → red → green → (refactor →) done`, which
carries no self-loops) refuses with an illegal-transition error, while every
other mutating verb now converges to a `Result.NoOp` at exit 0.

It sits behind an explicit allowlist entry in
`internal/policies/verb_result_noop_invariant.go`, so the convergence chokepoint
reports green while this hole stays open. Closing it means either converting the
verb or replacing that entry with a by-design rationale.

## Why it matters

The phase ladder is not ordinary state — it is the *evidence* that the test came
before the code, read back through `aiwf history`. Two properties make the
same-phase decision genuinely open rather than a mechanical repeat of the
status-transition case settled in ADR-0036:

- **The `--tests` payload.** Phase promotion accepts test metrics that write an
  `aiwf-tests` trailer. Refusal currently makes "re-record a phase with fresh
  metrics" impossible. A bare NoOp keeps it impossible but silent, and forecloses
  the competing reading — that a repeat phase call carrying new metrics should
  record a fresh observation — without deciding it. A conversion therefore needs
  the same "and nothing else is changing" gating the status-promote guard uses
  for resolver flags, chosen deliberately rather than by omission.
- **Diagnostics on a chronology surface.** Phase promotes fire live and ungated
  by design. A duplicated `--phase green` that silently exits 0 removes the
  "your model of the state is wrong" signal from a scripted or agent-driven
  ladder. Integrity is not at stake — the `acs-tdd-audit` rule enforces
  `met` requires `tdd_phase: done` independently, and `--force` does not bypass
  it — so this is a diagnostics tradeoff, on the surface where being loud is
  worth the most.

The surrounding surfaces look narrow: the legal-workflow spec carries AC-kind
cells with `tdd_phase` preconditions, the negative-driver's phase probe selects a
deliberately different phase, and the stresstest walk does not drive phases at
all. That reads as "no oracle cascade," but the status-transition case presented
the same way and turned into a four-surface change plus a kernel ADR — so the
scan wants redoing, not assuming.

## Resolution shape

Either (a) convert with an explicit metrics-carve-out and record the phase-FSM
decision alongside the status one, or (b) keep the refusal and rewrite the
allowlist entry to state the by-design reason. Not "leave the entry as-is."
