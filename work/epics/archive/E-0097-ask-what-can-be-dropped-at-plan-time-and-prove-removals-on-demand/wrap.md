# Epic wrap — E-0097

**Date:** 2026-09-27
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0097-ask-what-can-be-dropped-at-plan-time-and-prove-removals-on-demand

## Milestones delivered

- M-0356 — Ask for the smallest version at plan time (merged fe919c833)
- M-0357 — Ship the on-demand subtraction skill (merged 7e1e3b6df)
- M-0358 — Wire the subtraction lens into the wrap rituals (merged a9e437542)

## Changelog entry

### Added — E-0097: ask what a change can do without, at plan time and on demand

- `aiwfx-plan-milestones` can now remove work, not only reshape it. A candidate no
  success criterion requires is dropped, where before it could only be kept, split
  or folded into a sibling, and the cuts are put in front of you, with a yes asked
  for, before any id is allocated.
- New `wf-trim` skill in the `wf-*` rituals: ask of one diff or one named
  unit whether the change needs everything it adds. It looks for duplicated jobs,
  logic that compresses, guards no caller reaches and tests no break needs, and
  settles every removal it proposes by a command: breaking what the removed thing
  protected turns something red, or a guard is shown unreachable; a cut that
  changes no result must pass a differential test. It reports, and applies nothing
  until approved. It needs no aiwf verb or configuration, and a per-stack table
  names the mutation harness, coverage profile and clone detector for Go, Python,
  JavaScript/TypeScript and the JVM.
- `aiwfx-wrap-milestone` and `wf-patch` now call `wf-trim` as a review lens. In the
  milestone wrap it replaces the shape questions the wrap asked inline — deletions,
  same-outcome tests, compression and over-guarding — and the wrap keeps one
  question of its own: what the change obliges later changes to do, each obligation
  named with its owner and what retires it, recorded under the milestone spec's
  `## Reviewer notes`. In `wf-patch`, which asked none of those questions (G-0662),
  it is a new lens that runs on a
  reviewed patch changing code, tests included; a patch that changes only prose or
  configuration states the skip at the commit gate. In both rituals the lens applies
  nothing: its proposals reach you once all reviews have returned.

## Summary

The epic put the subtraction question at the two moments it pays: before work is
proposed, where the milestone sizing rule now drops what no success criterion
requires, and on demand over a change, where the new `wf-trim` skill settles each
removal it proposes by a command. The wrap rituals call the skill rather than
restating its procedure, so the patch ritual reaches the shape questions for the
first time (G-0662). Two other plan-time additions were measured under M-0356 and
cut: a scope offer in epic planning, which produced nothing the always-on rule did
not, and that always-on rule itself, which removed nothing on its own.

## ADRs ratified

- none

## Decisions captured

- none — each choice made during the epic is recorded in the spec it governs: the
  cut scope offer and always-on rule in the epic's Out of scope, the cancelled table
  check in M-0357, and the patch lens's threshold under M-0358's Decisions made
  during implementation.

## Follow-ups carried forward

- G-0723 — `wf-trim` step 2 leaves step 6 unstated for a change with neither logic
  nor tests.

## Doc findings

Clean. The change-set touches no file under `docs/`, `README.md` or
`CONTRIBUTING.md`; every relative link in its changed markdown resolves outside the
generated `ROADMAP.md`; every `aiwf` invocation it adds names a real verb.

## Handoff

Every success criterion is met; the skill and both wirings reach consumers with the
next release. Deliberately left open:

- G-0723, and three wording issues in `wf-trim` recorded under M-0357's Reviewer
  notes, wait for the next edit to the skill.
- `wf-patch` asks for no obligation's owner or retirement; the lens reports
  obligations as findings, and the question stays with the milestone wrap
  (M-0358's Reviewer notes).
- M-0358's wrap trial replayed M-0321, which is on `main`, and all three lenses
  found that rows in `spec.Rules()` are never checked against the applicability
  table: a row at a pair declared inapplicable, or at a sub-kind missing from
  `workflowKinds()` in `internal/policies/`, passes every M-0321 policy. It was
  measured at M-0321's own head and needs a reproduction on `main` before it
  becomes a gap.
- Build-time prevention — the commit-gate question and attrition in review
  dispositions — stays out of scope, to revisit once this epic's step has been
  observed.
