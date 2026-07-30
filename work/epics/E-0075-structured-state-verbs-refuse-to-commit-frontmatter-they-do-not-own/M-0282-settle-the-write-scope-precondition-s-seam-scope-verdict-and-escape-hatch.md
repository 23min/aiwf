---
id: M-0282
title: Settle the write-scope precondition's seam, scope, verdict and escape hatch
status: draft
parent: E-0075
tdd: none
acs:
    - id: AC-1
      title: ADR records where the precondition runs relative to the same-state check
      status: open
    - id: AC-2
      title: ADR records whether the guard is entity-scoped or committed-path-scoped
      status: open
    - id: AC-3
      title: ADR records refuse-or-warn, weighed against the illegal-transition escape
      status: open
    - id: AC-4
      title: ADR records whether an escape hatch exists and what it costs
      status: open
---

## Goal

Settle, in one ADR, the four decisions that determine what the write-scope
precondition is: where it runs, what it compares, whether it refuses, and
whether it can be overridden.

## Context

E-0075 identifies two distinct laundering mechanisms and shows that a guard
closing one leaves the other open. Every later milestone in this epic implements
against answers this one produces, and E-0074 waits on the first of them,
because `PromoteACPhase` writes frontmatter and so falls inside this epic's
route list.

Two facts about the current code bound the decisions. `tree.Tree` carries
`Root`, so any verb holding a loaded tree can already reach HEAD — the seam
choice is not constrained by plumbing. And `planEntityWrite` covers seven call
sites across six files, while `promote`, `move`, `rename`, `reallocate`,
`set-priority`, `set-area` and `edit-body` build their plans directly — so there
is no existing single-entity seam to simply extend, and "put it at the shared
seam" names a seam that has yet to be created.

## Approach

One ADR, four decisions, each recorded with the reasoning that produced it
rather than the verdict alone. Order matters: the seam decision constrains what
the scope decision can mean, and the verdict decision is what makes the
escape-hatch question live at all.

`checkStagedConflict` is read first, as the precedent E-0075 names — its message
shape, its position in the one function every write route passes through, and
its prefix-matching treatment of paths nested under a directory move.

## Acceptance criteria

### AC-1 — ADR records where the precondition runs relative to the same-state check

The ADR names the seam and states why. The choice is consequential rather than
stylistic: a guard at `verb.Apply` catches the empty-diff case, which does
produce a plan, but cannot catch a false NoOp, because the same-state guard
returns from the verb body before any plan exists. A prelude seam reaches both,
at the cost of having to be reached by every route — which is what the epic's
last milestone makes mechanical.

Evidence: a structural assertion that the ADR's `## Decision` section names a
seam, and that `## Consequences` states which of the two misbehaviors — a false
"already set" NoOp, an empty-diff commit — the chosen seam reaches.

### AC-2 — ADR records whether the guard is entity-scoped or committed-path-scoped

The nested case forces this: a guard comparing only the verb's named entity
misses a nested milestone's frontmatter riding along inside a parent epic's
directory move, and no verb names those entities.

Evidence: a structural assertion that `## Decision` records one of the two
scopes and that the nested-path case is addressed in that decision's own text,
not merely listed elsewhere in the document.

### AC-3 — ADR records refuse-or-warn, weighed against the illegal-transition escape

The weighing is against a laundered `status` on a path-changing route, not
against a laundered `priority`. Refusing blocks a workflow that currently
succeeds; permitting one lets a blocking check be bypassed.

Evidence: a structural assertion that `## Decision` records refuse or warn, and
that its reasoning cites the illegal-transition escape rather than only the
misattribution case.

### AC-4 — ADR records whether an escape hatch exists and what it costs

Not a question of reusing an existing lever: of the routes in this epic's scope,
only `promote` and `cancel` expose `--force`. Extending it is a surface
expansion carrying a completion-drift obligation, and the flag already means
several distinct things across the CLI.

Evidence: a structural assertion that `## Decision` records whether a hatch
exists; where one does, that the same section names the flag and the
completion-wiring obligation it incurs.

## Constraints

- Decision is decision. The ADR records the choice, not a schedule for acting on
  it — no gate language in the body, per this repo's rule on authoring an ADR.
- Each decision carries its reasoning, not only its verdict. The epic's
  *Open questions* table states what each one must weigh.
- The third decision is weighed against the illegal-FSM-transition escape. A
  laundered `priority` misleads a reader; a laundered `status` on a path-changing
  route defeats a blocking check, and that is the load-bearing case.
- Whatever the ADR decides must not be readable as fixing the FSM walker's
  rename-plus-status blind spot. G-0475 stays open on its own terms.

## Design notes

- The epic's fifth open question — whether each multi-entity sweep is in or
  out — is assigned to M-0284 rather than to this ADR, but it is partly derived
  from AC-2. Under committed-path scope a sweep's paths *are* the commit's
  paths, so the sweeps are in by construction; under entity scope each needs an
  individual call. The scope decision therefore determines whether that question
  is answered or merely posed.
- `checkStagedConflict` already solves the nested-path problem for the staged
  case, by prefix-matching staged paths against an `OpMove`'s source and
  destination rather than walking the filesystem. Whatever the ADR decides about
  scope, that is the cheapest available shape and the one whose message the
  operator has already seen.
- The comparison degrades when there is no HEAD version to compare against.
  `edit-body` already contains both answers — bless mode refuses a
  never-committed entity, explicit mode proceeds — so this is an implementation
  note rather than a fifth decision.

## Out of scope

- Implementing the precondition. This milestone produces a decision record.
- The per-sweep in-or-out calls, which belong to M-0284.
- The FSM history walker's rename-plus-status blind spot (G-0475).
- The after-the-fact detection rule (G-0480), which is a companion to the
  precondition rather than an alternative to it.

## Dependencies

- None. First milestone in E-0075.

## References

- E-0075 — the parent epic, whose *Open questions* table is this ADR's agenda
- G-0466 — a verb commits frontmatter it does not own
- G-0463 — the `edit-body --body-file` instance of the same question
- G-0475 — the FSM walker's rename-plus-status blind spot
- G-0480 — after-the-fact detection of laundering already in history
- ADR-0036 — same-status FSM transitions converge to NoOp, not refusal
- E-0074 — waits on this milestone's first decision
- `internal/verb/apply.go` — `checkStagedConflict`, the precedent to extend

