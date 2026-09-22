---
id: ADR-0050
title: Converge repeated TDD phases without discarding metrics
status: proposed
---
> **Date:** 2026-09-22 · **Decided by:** Peter Bruinsma

## Context

ADR-0036 makes a same-status request converge without adding a history event.
TDD phase promotion also names a desired state, but can carry test metrics that
belong to a phase event. A same-phase success must not silently discard that
payload or manufacture another transition in the test chronology.

G-0458 names this remaining convergence decision. M-0318 needs an explicit
outcome for a phase's self-target, rather than leaving that coordinate ambiguous.

Keeping every same-phase request illegal preserves a loud diagnostic for retries,
but makes the phase command an exception to the same-state convention. Accepting
metrics on a NoOp would lose evidence behind a success response. Appending a new
phase event for a repeat would instead turn state promotion into event recording.

## Decision

A request to promote an AC to its current recognized TDD phase converges to NoOp
when no test-metrics payload is supplied: exit 0, a message naming the phase,
no plan, no file changes, and no commit.

A same-phase request carrying test metrics is refused with a diagnostic that says
metrics require a phase change and that omitting them permits convergence.
Explicit zero counts still count as a supplied payload. Empty or whitespace-only
metrics text retains the CLI's existing meaning of no payload.

Resolve the AC and check its parent milestone against the committed file before
claiming convergence. An absent or unrecognized phase cannot converge. These
rules apply before the force override: force neither creates a redundant event
nor permits metrics to be discarded on a same-phase request.

Real phase changes continue through the FSM and existing metrics-trailer path.
Self-loops are not added to the FSM. Audit-only phase promotion is a separate
recording operation and is outside this decision.

The legal-workflow table distinguishes metrics-bearing requests explicitly:
`self.tests == ""` denotes no supplied payload, and `self.tests non-empty`
denotes one. The evaluator reads this request context rather than a field stored
on the entity. Each recognized phase has separate self-target NoOp and refusal
rows, and the drivers exercise their respective requests.

## Consequences

Ordinary phase retries are idempotent without inventing chronology. A caller that
supplies metrics must make a real phase transition; this command provides no way
to attach another metrics observation to an already recorded phase.

The convergence exemption for phase promotion is removed. The ordinary NoOp and
claim-scope policies cover it, including protection against uncommitted milestone
content. E-0089 and M-0318 own G-0458's implementation; E-0074 retains the separate
event-recording convergence questions.

## Validation

Verb tests cover recognized phases, forced repeats, explicit zero and nonzero
metrics, real advances with metrics, invalid or absent phases, and dirty milestone
content. CLI tests require exit 0 only for bare repeats, refusal for supplied
metrics, and unchanged HEAD and project files in both cases. Table tests require
exactly one applicable self-target row for each phase and metrics-presence case.
Mutation probes must expose bypassed metrics refusal, bypassed claim guards,
missing phase validation, and loss of explicit-zero payload presence.

## References

- ADR-0036 — same-status convergence
- ADR-0038 — committed-content checks before a convergence claim
- G-0458 — the TDD same-phase decision
- M-0318 — target and outcome correctness
- E-0089 — the legal-workflow table
- E-0074 — event-recording convergence
