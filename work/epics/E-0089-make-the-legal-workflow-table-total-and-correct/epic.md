---
id: E-0089
title: Make the legal-workflow table total and correct
status: proposed
---
## Goal

Make the legal-workflow spec table total and correct, so that it can be read — and
rendered — as the authoritative answer to *what states exist, what moves between
them, and what is refused*, with no coordinate left silent and no declared outcome
that the kernel contradicts.

## Context

ADR-0011's three-pass methodology produced the spec table and M-0123 reconciled it
against the implementation, with a bidirectional drift policy closing spec and impl
against each other. That machinery works: the drift arms catch a new state, a new
verb, an unresolvable finding code, a schema violation.

What it does not catch is a coordinate nobody wrote, or a declared outcome that is
simply wrong. Enumerating every (Kind, FromState, legality verb) coordinate against
`spec.Rules()` on 2026-08-24 measured 99 coordinates, of which 49 carry a cell and
50 do not — and nothing in the tree distinguishes a coordinate left silent on
purpose from one nobody considered. `promote` is complete at 33 of 33; every hole
is `cancel` (21) or `authorize` (29).

Worse than the holes is what the declared cells say. All 15 terminal-state
`promote` coordinates are declared illegal, while M-0281/AC-1 makes a promote to
the entity's current status a NoOp that exits 0 (G-0631). Two cells declare
check-time rejection where the kernel refuses at verb time (G-0166). Two more cite
a finding code that was retired (G-0417). In each case the spec kept an answer the
kernel had moved past, and the drift policy did not notice because its arms compare
coordinates, kinds, states and codes — never a declared outcome against what the
verb returns.

D-0077 settles the shape of the fix. The target joins the cell key, so a NoOp is
expressible and an illegal transition from the same origin stays separately
expressible; `authorize` leaves the cell table for `GlobalRules()`, where its
substantive rules already live; and applicability becomes a kind-by-verb fact
declared once, so totality is defined over rows where the verb means something.

G-0160 reached the same conclusion independently from the coverage side: its fix
outline names splitting cells per target as the way to let the drift policy compare
claimed targets against the FSM's allowed edges. D-0077 ratifies that option, and
this epic implements it.

## Scope

- **Split every cell by target**, adding the target to `Rule` and to the enforced
  uniqueness key. Derive it mechanically from `entity.transitions` where the cell
  is legal; rule the illegal and NoOp rows by hand.
- **Move `authorize` out of the cell table** into `GlobalRules()`, and sweep the
  stale entries already sitting there while the file is open.
- **Declare kind-by-verb applicability once**, policed total over every pair, and
  define cell-table totality over applicable rows.
- **Re-key the positive and negative coverage drivers** so per-cell coverage becomes
  exhaustive by construction, and reconcile the rejection-layer axis against where
  the kernel actually refuses.
- **Decide whether to render the table** as a generated reference, and ship it if
  the decision says so.

## Out of scope

- **The `tdd_phase` same-phase question.** G-0458 holds it, and it is a genuine open
  design question rather than a mechanical repeat: phase promotion carries a
  `--tests` payload and reads back as the evidence that the test came first. This
  epic makes a NoOp expressible in the table; it does not decide that case.
- **Widening the table past status-transition verbs.** The path-changing verbs have
  no legality model, and whether they should is a separate question that E-0088's
  findings inform.
- **The two legal-workflow catalogs under `docs/design/`.** Their disposition is its
  own work; this epic changes what the table says, not what the working papers hold.
- **Retiring the drift policy's existing arms.** They stay; this epic adds what they
  cannot see.

## Constraints

- **D-0077 is the specification.** A claim here that D-0077 does not carry is a
  defect in one of the two.
- **Every hole is closed by a ruling, not by a default.** A cell written because the
  grid demanded one, without an argument for its outcome, is worse than the silence
  it replaced — it reads as a decision.
- **The mechanical pass is not the deliverable.** Deriving targets from
  `entity.transitions` fills the legal rows; the residue is where the work is, and
  it is read, not generated.
- **The kernel is the tiebreak for behavior, the human for intent.** Where the table
  and the kernel disagree about what a verb does, the kernel is right and the table
  is corrected. Where they disagree about what a verb *should* do, it is a decision.
- **Totality is enforced, not asserted.** A coordinate with no cell fails a policy;
  it does not merely go unmentioned in a review.

## Success criteria

Observable at epic close. Milestone acceptance criteria carry the mechanical bar.

- [ ] Every applicable (Kind, FromState, Verb) coordinate carries at least one cell,
      enforced by a policy that fails when one does not.
- [ ] Every cell names its target, and the enforced uniqueness key includes it.
- [ ] The kind-by-verb applicability table is total over every pair, enforced.
- [ ] No declared outcome contradicts what the verb returns at that coordinate,
      demonstrated by a check that compares the two rather than by inspection.
- [ ] `authorize` appears in no cell; its kind restriction is expressed in
      `GlobalRules()` and still fails when removed.
- [ ] G-0631, G-0160, G-0417 and G-0166 are each closed or explicitly re-scoped with
      the reason.
- [ ] The render decision is recorded, and the render ships if it was taken.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Does a rendered reference ship from this table, or does the table stay a code-side artifact? | no | Decided in the milestone that would build it, before the code lands. |
| Do the two check-time cells keep that axis, or does the spec follow the kernel to verb-time? | yes, for the driver milestone | Answered by that milestone against G-0166's evidence. |
| How large does the table get once cells split by target? | no | Measured by the first milestone; the estimate of roughly 120–180 is not evidence. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| The mechanical pass silently fills rows that deserved a ruling | high | The pass emits only cells derivable from `entity.transitions`; every other row is listed for hand-ruling and the count is recorded. |
| Splitting cells breaks the coverage drivers in ways that hide coverage loss | high | The driver milestone re-measures per-cell coverage before and after, and totality becomes a policy rather than a property of the driver. |
| The table grows enough that reading it stops being useful | med | The render exists for readers; the table is for machines. If the render is declined, the size argument is recorded against that decision. |
| Closing four gaps in one epic lets one of them be quietly dropped | med | Each named gap is a success criterion, closed or re-scoped with a reason. |

## Milestones

Sequenced so the key change lands first, since everything else is expressed in
terms of it, and the render lands last, when there is a total table to render.

- `M-0318` — split every cell by target and correct the outcomes the kernel
  contradicts, establishing the key everything else is expressed in ·
  depends on: —
- `M-0319` — move `authorize` to `GlobalRules()` and sweep the stale entries
  already there; independent of the key change and runnable in either order ·
  depends on: —
- `M-0320` — re-key the coverage drivers and settle whether each cell's rejection
  layer names where the kernel actually refuses · depends on: `M-0318`
- `M-0321` — declare kind-by-verb applicability and make a missing cell at an
  applicable coordinate a policy failure · depends on: `M-0318`, `M-0319`
- `M-0322` — decide the rendered legality reference and ship it if taken;
  sequenced last so a render cannot publish holes · depends on: `M-0321`

## References

- D-0077 — the specification: key the table by transition, not by coordinate
- ADR-0011 — the three-pass methodology that produced the table
- ADR-0013 — global rules as the home for preconditions carrying no cell coordinate
- D-0007 — the `authorize` kind restriction being moved
- M-0281 — the same-state NoOp convention the table does not yet express
- G-0631 — terminal-state promote declared illegal where the kernel returns NoOp
- G-0160 — per-edge drift unpoliced; its fix outline names this epic's approach
- G-0417 — stale finding-code entries in the table `authorize` moves into
- G-0166 — cells declaring check-time rejection that the kernel refuses at verb time
- G-0458 — the `tdd_phase` same-phase question, out of scope and left with its owner
- `internal/workflows/spec/` — the table; `internal/policies/m0123*`, `m0124*`,
  `m0125*` — the drift and coverage machinery
