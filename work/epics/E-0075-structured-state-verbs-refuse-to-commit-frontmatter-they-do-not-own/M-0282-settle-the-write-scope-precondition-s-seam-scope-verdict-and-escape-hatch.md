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

`planEntityWrite` covers seven call sites across five files, while `promote`,
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

The mechanical evidence for all five is one structural assertion,
`PolicyM0282ADRWriteScopeDecisions`: each named `### ` subsection exists under
`## Decision` with prose beneath it, `## Consequences` is non-empty, and the ADR
is `accepted`. Placement is the substance — a heading carrying the same words
elsewhere in the document does not satisfy it.

It asserts shape, not that a subsection records a real decision. That was
attempted with keyword matching and does not work at any tightening, because a
deferral names every term a decision would name: "whether it refuses or warns is
deferred" contains "refuse". The distinction is meaning rather than vocabulary,
so it is a review judgment and sits at the human gate, where CLAUDE.md puts
judgment classes no check can cover. The policy's own test pins that limit, so a
later attempt to mechanise it has to change the test deliberately rather than
drift into overstating what the criteria are backed by.

### AC-1 — ADR records where the precondition runs relative to the same-state check

The ADR names the seam and states why. The choice is consequential rather than
stylistic: a guard at `verb.Apply` catches the empty-diff case, which does
produce a plan, but cannot catch a false NoOp, because the same-state guard
returns from the verb body before any plan exists. A prelude seam reaches both,
at the cost of having to be reached by every route — which is what the epic's
last milestone makes mechanical.

Evidence: the `### Seam` subsection exists under `## Decision` with prose
beneath it. Which seam it names, and whether that seam reaches both measured
misbehaviors, is read at review.

### AC-2 — ADR records whether the guard is entity-scoped or committed-path-scoped

This is the *path* axis — which files the guard compares. AC-5 is the
orthogonal *field* axis, which parts of each file. Both need an answer, and
answering one does not imply the other.

The nested case forces this one: a guard comparing only the verb's named entity
misses a nested milestone's frontmatter riding along inside a parent epic's
directory move, and no verb names those entities.

Evidence: the `### Path scope` subsection exists under `## Decision` with prose
beneath it. Whether it addresses the nested case is read at review.

### AC-3 — ADR records refuse-or-warn, weighed against the illegal-transition escape

The weighing is against a laundered `status` on a path-changing route, not
against a laundered `priority`. Refusing blocks a workflow that currently
succeeds; permitting one lets a blocking check be bypassed.

AC-5's answer feeds directly into this one: under a whole-file field scope the
guard fires during the ordinary bless workflow, so "refuse" is a far larger
behavioral change than it is under a frontmatter-only scope. The two are decided
together or not at all.

Evidence: the `### Verdict` subsection exists under `## Decision` with prose
beneath it. Whether its reasoning weighs the illegal-transition escape rather
than only misattribution is read at review.

### AC-4 — ADR records whether an escape hatch exists and what it costs

Not a question of reusing a consistent lever. Measured, `--force` already appears
on `promote`, `cancel`, `contract bind` and `contract recipe install` among the
routes in scope, and on `authorize` and `add` elsewhere — carrying *sovereign
act* in one place and *born-complete-body bypass* in another. Extending it would
overload an already ambiguous flag and carry a completion-drift obligation on
every route that gained it.

Evidence: the `### Escape hatch` subsection exists under `## Decision` with
prose beneath it. Whether it settles the question, and at what stated cost, is
read at review.

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

Evidence: the `### Field scope` subsection exists under `## Decision` with prose
beneath it, and `## Consequences` is non-empty. Whether the cost to the
review-before-commit window is actually weighed there is read at review.

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
- The guard's remaining mechanics — how the compared path set is finally derived,
  whether carry-along substitution is adopted, the `ExitUsage` change's blast
  radius, and which tests break. Each was reached for here and each turned out to
  need a prototype, so ADR-0038 records them as deferred and M-0283 settles them
  against measurement.
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
f642b502e (accept), 53d6abd50 (canonical subsection anchors), ff946fce5
(narrowed after review) · every test in the package green

`PolicyM0282ADRWriteScopeDecisions` (commits c612e5e65, 56688438d, and the
narrowing that followed) asserts that each decision has its own `### `
subsection under `## Decision` with prose beneath it, that `## Consequences` is
non-empty, and that the ADR is `accepted`. Structural rather than substring: a
heading must sit under `## Decision` specifically, so the same words elsewhere
in the document do not satisfy it. One firing fixture per failure mode —
including the present-but-empty arms and both non-`accepted` statuses — plus a
clean-fixture negative control and the loader-miss arm.

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
every mutating verb refuses cleanly on an unparseable entity, and the recovery is
complete already: the `load-error` hint names the hand-fix, and the
`provenance-untrailered-entity-commit` hint names acknowledging the hand-commit
that lands it. No new surface was needed.

## Decisions made during implementation

- **ADR-0038** — the milestone's deliverable, recording the seam, path scope,
  field scope, verdict and escape hatch. Accepted, then narrowed twice under
  review: carry-along substitution was removed to M-0283 after prototyping
  broke it, and the `ExitUsage` correction was demoted from decision to open
  question once it turned out to need a change in `internal/cli/cliutil`.
- **The mechanical evidence was narrowed rather than tightened.** Recorded in
  `## Reviewer notes` below, and in the policy's own doc comment. No separate
  decision entity: it settles how one policy is written, not a project-wide
  convention.
- **The epic's milestone split changed** — M-0283 absorbs the spike and the
  nested-path vector; M-0284 becomes the claim-side seam plus the sweeps'
  recorded calls. Recorded in E-0075's *Milestones* section.

## Validation

`make ci` green — race suite, lint, the profile-driven gates and
`aiwf doctor --self-check` (29 steps). `make coverage-gate` green, including the
diff-scoped branch-coverage audit and the firing-fixture meta-gate.
`aiwf check --since origin/main` reports no findings.

`internal/policies` passes in full. The M-0282 policy's own tests cover the live
ADR, a clean negative control, one firing fixture per failure mode — including
both present-but-empty arms and both non-`accepted` statuses — the loader-miss
arm, and the test pinning that a deferring document passes.

One flake seen and diagnosed, not caused here:
`TestRun_FixtureRejected_OneFailingValid` in `internal/contractverify` failed one
`make ci` run and passed in isolation on this branch and on trunk. The test
writes a `#!/bin/sh` validator at mode `0o755` and execs it, which races to
`ETXTBSY` under parallel load — G-0462's first mechanism, tracked and scheduled
in E-0077.

## Deferrals

Four questions ADR-0038 records as deferred, all assigned to M-0283 and answered
there from a prototype rather than by argument: how the compared path set is
derived; whether carry-along substitution is adopted and under what corrections;
what the `ExitUsage` change costs across `internal/cli/cliutil`; and which
existing tests break.

Three defects surfaced by this milestone's reviews, each pre-existing, none in
this epic's scope, all filed rather than absorbed:

- **G-0486** — directory moves dereference symlinks and force mode `100644`.
  Measured on a clean tree, durable in a fresh clone, `aiwf check` 0 errors.
- **G-0487** — `assume-unchanged` / `skip-worktree` / sparse checkout hide a
  dirty path from both of `DirtyPaths`' queries. Bounds M-0283's guard directly.
- **G-0488** — the loader's `area` normalization rides into the next verb's
  commit, with no divergence anywhere.

## Reviewer notes

**Two independent review rounds, four reviewers, and the ADR changed
substantially in both.** Round one attacked the first draft: it found that a
comparison sited where `gatherCommitOps` runs reads the verb's own
freshly-written bytes, that `edit-body` bless mode would refuse itself under the
guard it recommends as the escape, and that frontmatter-only scope at the
claim-side seam contradicts five sites that already compare a second surface.
Round two attacked the amendment: a refinement adopted from round one —
substituting HEAD's blob for paths no verb named — was prototyped and found to
duplicate subtrees under `retitle` / `move` / `reallocate`, drop a milestone from
a `rewidth --apply` commit while reporting success, and leave flat-file moves
uncovered. A separate fact-check found four claims that were false about the
code, three of them inherited from a reviewer rather than measured here.

**Every defect in both rounds was an implementation discovery, not a reading
one.** That is the durable lesson and it reshaped the epic: M-0283 now leads with
a throwaway prototype and a scenario matrix, and this ADR records as *deferred*
the questions that a prototype has to answer. The alternative — continuing to
settle mechanics in prose — was measurably producing a new defect per round.

**AC-2 and AC-3 were promoted to `met` before their evidence existed.** Their
bodies each state a two-part obligation (path scope *and* the nested case;
refuse *and* the illegal-transition weighing) and the policy asserted only the
first half of each; stripping either reasoning sentence from the ADR produced
zero violations. The policy now requires both conjuncts, and a purpose-built
"decides nothing" ADR no longer passes. The premature promote itself is not
reversible: the AC FSM allows `met → deferred | cancelled` only, both
removal-class, so demoting would claim the criteria are off the contract rather
than that their evidence arrived late. They stay `met` with the gap recorded
here instead.

**The mechanical evidence pins recording, not implementability.** An ADR can
satisfy all five criteria while deciding something that cannot be built — which
is exactly what round one found. M-0283/AC-4 is where implementability becomes
the evidence, because that is where a prototype exists to test it against.

**A third round found the strengthened assertion still inadequate, and the fix
was to narrow the claim rather than tighten the match.** Measured: a document
deferring every question passed with zero violations, in the same voice this ADR
uses for what it defers. Keyword matching cannot separate a decision from a
deferral at any tightening, because a deferral names every term a decision would
name. The policy now asserts shape, placement and status only, its documentation
says so, and a test pins the limit by asserting that a deferring document
*passes* — so a later attempt to mechanise the judgment has to change that test
deliberately.

The general lesson is worth more than the instance: the acceptance criteria
reached for mechanical evidence of something inherently non-mechanical. The
repo's own rule asks for "a structural assertion scoped to a named markdown
section", which is what now exists; the bar of proving a decision was *made* was
invented here and could not be cleared. Where mechanical evidence genuinely fits
this epic is M-0283, whose matrix crosses an enumerable path-state space with
verb class — completeness of that table is assertable in a way a document's
meaning is not.
