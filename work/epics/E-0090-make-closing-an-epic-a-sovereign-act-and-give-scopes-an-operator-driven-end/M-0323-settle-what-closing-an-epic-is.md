---
id: M-0323
title: Settle what closing an epic is
status: draft
parent: E-0090
tdd: none
acs:
    - id: AC-1
      title: An accepted ADR exists and E-0090's ADRs produced section cites it by id
      status: open
    - id: AC-2
      title: E-0090's Open questions table routes every row to that ADR
      status: open
---
## Goal

Settle, in one ADR, what closing an epic is: who may declare it on each terminal edge, and how an operator-driven scope end coexists with the automatic one.

## Context

The kernel gates one sovereign transition, epic `proposed → active`, and neither edge that closes an epic. The gate is promote-only: `requireHumanActorForSovereignAct` has a single call site, so `aiwf cancel` never consults the closed set. Separately, a scope's only exit is the terminal promote or cancel of its own entity, which stamps `aiwf-scope-ends` as a side effect.

Three questions have to be answered before either code milestone can be built, and answering them apart would fork one decision across two records.

## Acceptance criteria

### AC-1 — An accepted ADR exists and E-0090's ADRs produced section cites it by id

The ADR resolves through the loader at status `accepted`, and the epic's `## ADRs produced` section names that id. The check is a relationship between two artifacts: it fails if the ADR is missing, is not accepted, or the epic cites a different id.

It deliberately does not assert that the ADR answers the questions *well*. That is content correctness over prose, which this repo holds at review; asserting it with a phrase match would pin a reading that rewording breaks.

### AC-2 — E-0090's Open questions table routes every row to that ADR

Every row of the epic's `## Open questions` table names the AC-1 ADR as its resolution path. Fails if a row still points elsewhere, or if a question was added after the ADR landed and left unrouted.

## Constraints

- The ADR records the choice, not the schedule for acting on it. No gate language in the body.
- The end-mode targeting answer must be consistent with the provenance model's existing rule that a verb under multiple active scopes picks the most-recently-opened.

## Design notes

The three questions carried from the epic:

1. Does the end mode target the most-recently-opened active scope, mirroring `--pause` / `--resume`, or every active scope on the entity, mirroring today's auto-end? G-0460 establishes that multiple simultaneously-active scopes are legal and intended, so the two existing answers genuinely disagree.
2. Is cancelling an epic a sovereign act? The epic's scope already commits to yes; the ADR records that and its reasoning.
3. What undoes an end? `ended` is terminal in the scope FSM and re-authorizing opens a fresh scope rather than reviving the old one — the ADR states whether that is the whole answer, per the kernel's what-undoes-this rule for a new verb surface.

ADR-0040 constrains the shape of any answer to question 2: prevention belongs at the verb route, so a widened closed set arrives with its call site rather than before it.

## Out of scope

- Any code change. This milestone produces a decision record.
- Whether the automatic scope-end survives. The epic already decided it does.

## Dependencies

- None. This is the epic's first milestone; both others depend on it.
