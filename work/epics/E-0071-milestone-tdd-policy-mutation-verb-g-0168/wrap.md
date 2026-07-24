# Epic wrap — E-0071

**Date:** 2026-07-24
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0071-milestone-tdd-policy-mutation-verb-g-0168
**Merge commit:** 691d1e0d

## Milestones delivered

- M-0277 — Add the aiwf milestone tdd policy-mutation verb (merged 5f5f7f71)

## Summary

E-0071 gave a milestone's `tdd:` policy a first-class post-creation mutation
verb — `aiwf milestone tdd <M-id> --policy none|advisory|required` — closing the
`tdd:` slice of G-0168's set-at-create verb-chokepoint hole. Gating is
uniform-ordinary per D-0048: any authorized actor may flip the policy in either
direction with no `--force`, and there is no directional or sovereign carve-out.
A flip to `required` that would strand an already-`met` AC lacking
`tdd_phase: done` is refused with an actionable hint naming the offending ACs,
rather than back-stamping a phase onto already-passed work. The verb ships fully
discoverable (root banner, `--policy` completion, `aiwf-add` skill mention) and
is wired into the `verb-sequence` stress-test walker as a milestone-only,
always-legal operation.

## ADRs ratified

- none

## Decisions captured

- none new — the epic is governed by D-0048 (accepted): the verb surface,
  uniform-ordinary gating, and the deferral of the three relation-field editors.

## Follow-ups carried forward

- G-0168 — remains **open**; only the `tdd:` slice is closed. The three
  relation-field editors (`discovered_in` / `relates_to` / `linked_adrs`) stay
  deferred per D-0048 until real friction appears.
- G-0442 — the set-at-transition amend pair (`addressed_by` / `superseded_by`),
  a separate sibling, deliberately out of scope here.
- G-0121 — the AC-composition invariant-fuzz the walker addition unblocks; a
  standalone milestone, deliberately not folded into the tdd-verb work.
- G-0166 — extending the RejectionLayer axis to model data-field-mutation
  rejections; M-0277's refuse-with-hint is a candidate cell the current
  `(Kind, FromState, Verb)` spec-table shape cannot yet model.

## Handoff

A milestone's TDD policy is now mutable post-creation through a trailered,
discoverable verb — no more hand-editing `tdd:` frontmatter. Deliberately left
open: the G-0168 relation-field editors, the G-0442 set-at-transition amends,
the G-0121 AC-composition invariant-fuzz, and the G-0166 spec-table extension.
Nothing blocks the next epic.

## Doc findings

doc-lint (scoped to the epic change-set): clean — no broken code references, no
stale CLI invocations, no broken links.
