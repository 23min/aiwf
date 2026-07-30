---
id: M-0282
title: Settle the write-scope precondition's seam, scope, verdict and escape hatch
status: in_progress
parent: E-0075
tdd: none
acs:
    - id: AC-1
      title: ADR records where the precondition runs relative to the same-state check
      status: met
    - id: AC-2
      title: ADR records whether the guard is entity-scoped or committed-path-scoped
      status: met
    - id: AC-3
      title: ADR records refuse-or-warn, weighed against the illegal-transition escape
      status: met
    - id: AC-4
      title: ADR records whether an escape hatch exists and what it costs
      status: met
    - id: AC-5
      title: ADR records whether the guard compares frontmatter only or the whole file
      status: met
---

## Goal

Settle, in one ADR, the five decisions that determine what the write-scope
precondition is: where it runs, which paths it compares, which parts of a file
it compares, whether it refuses, and whether it can be overridden.

## Context

E-0075 identifies two distinct laundering mechanisms and shows that a guard
closing one leaves the other open. Every later milestone in this epic implements
against answers this one produces, and E-0074 waits on the first of them,
because `PromoteACPhase` writes frontmatter and so falls inside this epic's
route list.

Three facts about the current code bound the decisions.

`tree.Tree` carries `Root`, so any verb holding a loaded tree can already reach
HEAD — the seam choice is not constrained by plumbing. `gitops.ReadFromHEAD` is
the primitive `edit-body` already uses for exactly this comparison, and
`gitops.BlobReader` is a batched cat-file reader exposing blob SHAs, so
comparing many paths does not cost one subprocess per path.

`planEntityWrite` covers seven call sites across six files, while `promote`,
`move`, `rename`, `reallocate`, `set-priority`, `set-area` and `edit-body` build
their plans directly — so there is no existing single-entity seam to simply
extend, and "put it at the shared seam" names a seam that has yet to be created.

And the laundering is whole-file rather than frontmatter-only. A serializing
verb reads the body off disk (`readBody`) and re-serializes it alongside the
frontmatter it computed, so an unblessed body edit rides into the commit exactly
as a hand-edited field does. Reproduced: a gap's body rewritten in the working
tree, then `aiwf set-priority` run, produced one commit carrying both the
priority change and the body rewrite under `aiwf-verb: set-priority`, with
`aiwf history` showing no `edit-body` event and `aiwf check` reporting no errors.
That is why the field-scope question below is a decision rather than an
implementation detail, and it is the reason this epic's title understates its
subject.

## Approach

One ADR, five decisions, each recorded with the reasoning that produced it
rather than the verdict alone. Order matters: the seam decision constrains what
the path-scope decision can mean, the field-scope decision sets how often the
guard fires at all, and the verdict decision is what makes the escape-hatch
question live.

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

This is the *path* axis — which files the guard compares. AC-5 is the
orthogonal *field* axis, which parts of each file. Both need an answer, and
answering one does not imply the other.

The nested case forces this one: a guard comparing only the verb's named entity
misses a nested milestone's frontmatter riding along inside a parent epic's
directory move, and no verb names those entities.

Evidence: a structural assertion that `## Decision` records one of the two path
scopes and that the nested-path case is addressed in that decision's own text,
not merely listed elsewhere in the document.

### AC-3 — ADR records refuse-or-warn, weighed against the illegal-transition escape

The weighing is against a laundered `status` on a path-changing route, not
against a laundered `priority`. Refusing blocks a workflow that currently
succeeds; permitting one lets a blocking check be bypassed.

AC-5's answer feeds directly into this one: under a whole-file field scope the
guard fires during the ordinary bless workflow, so "refuse" is a far larger
behavioral change than it is under a frontmatter-only scope. The two are decided
together or not at all.

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

### AC-5 — ADR records whether the guard compares frontmatter only or the whole file

The *field* axis, orthogonal to AC-2's path axis. E-0075 frames the defect as
frontmatter, but the mechanism is whole-file: a serializing verb re-serializes
the frontmatter it computed around a body it read off disk, so an unblessed body
edit lands in the commit under the verb's own trailer with no `edit-body` event
in `aiwf history`.

What makes this a genuine decision rather than a formality is the cost
asymmetry. A hand-edited frontmatter field is rare and nearly always a mistake.
An uncommitted body edit is *ordinary* — it is the normal mid-state of the
review-before-commit rhythm the shipped guidance recommends, where the body is
edited in the working tree, read as a real diff, then blessed with
`aiwf edit-body`. A whole-file guard refuses every structured-state verb run
inside that window.

That refusal is defensible: today the same window silently commits the body edit
under `aiwf-verb: promote`. But it makes the guard fire during a workflow the
project actively teaches, which is a different proposition from catching a rare
mistake, and it is the input the verdict decision in AC-3 has to weigh.

Evidence: a structural assertion that `## Decision` records a field scope, and
that `## Consequences` addresses what the chosen scope does to the bless
workflow — silence there would mean the cost was not weighed.

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


## Work log

All five acceptance criteria are satisfied by one deliverable — ADR-0038 — and
one mechanical assertion over it. They are logged together rather than
per-AC because no AC could be met without the others: the ADR is a single
document and the policy reads all five of its subsections in one pass.

### AC-1..AC-5 — ADR-0038 and its structural assertion

ADR-0038 accepted, five decisions recorded · commits 644cf9fe1 (add),
f642b502e (accept), 53d6abd50 (canonical subsection anchors) · tests 10/10

`PolicyM0282ADRWriteScopeDecisions` (commit c612e5e65) asserts each decision
appears in its own `### ` subsection under `## Decision` carrying a recorded
verdict, plus the seam's reach and the field scope's bless-workflow effect in
`## Consequences`. Structural rather than substring: the marker must sit
inside the named subsection, so prose mentioning "refuse" elsewhere in the
document does not satisfy the verdict decision. Eight firing fixtures, one per
failure mode, plus a clean-fixture negative case and the loader-miss arm.

The ADR resolves through `tree.Load` + `ByID` + `entity.Path` rather than a
path literal, so the assertion survives an archive sweep and a retitle.

Two decisions changed shape during the work rather than being transcribed
from the milestone spec:

- The seam is **two** seams, not one. A single top-of-verb helper would have
  to predict the paths a verb will touch, duplicating `gatherCommitOps`'
  recursive walk; siting the commit-side guard at `verb.Apply` — where that
  path set is already computed — removes the prediction, and the NoOp
  constructor covers the window `Apply` structurally cannot see.
- A fifth decision was added mid-milestone. The laundering is whole-file, not
  frontmatter-only: measured, an unblessed body edit rides into an unrelated
  verb's commit with no `edit-body` event in `aiwf history`. AC-5 and the
  epic's field-axis open question came from that measurement.

The escape-hatch decision also shed a proposed `aiwf repair` verb: measured,
every mutating verb refuses cleanly on an unparseable entity, and the recovery
the `load-error` hint already names (hand-fix, hand-commit, then
`aiwf acknowledge illegal`) is complete. No new surface was needed.
