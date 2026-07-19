# Epic wrap — E-0068

**Date:** 2026-07-19
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0068-mechanical-ac-milestone-completeness-guards
**Merge commit:** c1b29e1bab717cb8f5112bc058ba01686918492d

## Milestones delivered

- M-0267 — Relax acs-shape/tdd-phase to allow absent phase until AC met (merged f190e786)
- M-0268 — AC-completeness guards: zero-AC and empty-body promote refusals (merged 83e0e98c)

## Summary

Closed three places where the kernel depended on operator vigilance instead of a mechanical chokepoint for AC/milestone completeness discipline. M-0267 relaxed `acs-shape/tdd-phase` so an absent phase is legal on any AC until it reaches `met`, fixing the over-strict behavior that reddened the tree whenever a milestone strengthened `tdd: advisory → required`. M-0268 added two verb-time refusals (a milestone can no longer start `in_progress` with zero ACs, or with any AC whose body is a title-only stub) and two matching check-time findings (a warning for a zero-AC milestone reaching `done`, an error for an empty AC body persisting at `in_progress`/`done`, archive-scoped like every sibling rule in `acs.go`). Scope held to what every milestone listed above shipped — no scope drift.

## ADRs ratified

- none

## Decisions captured

- D-0039 — AC completeness guards: block empty start, warn at done, archive-scoped check (accepted pre-epic; the authoritative source for every guard's severity/override shape)
- D-0040 — AC-2 force override stays inert against AC-4's error (accepted during M-0268; a real interaction discovered mid-implementation between AC-2's `--force` and AC-4's new error-severity check, resolved by accepting the asymmetry rather than adding a bypass mechanism or downgrading AC-4)

## Follow-ups carried forward

- none

## Doc findings

`wf-doc-lint` (scoped to every file touched on `epic/E-0068-mechanical-ac-milestone-completeness-guards` since it diverged from `main`): no broken markdown links, no stale references to the epic's new finding codes or function names found across `docs/`. One pre-existing, non-blocking finding carried forward from M-0267's own wrap (unchanged, not introduced by this epic): `docs/pocv3/plans/acs-and-tdd-plan.md:206` states "`tdd_phase` is required when milestone `tdd: required`," which M-0267's own relaxation made stale. That document is explicitly `Status: proposal` — a pre-implementation planning artifact, not a live spec — so left as-is, consistent with the scoping decision M-0267's own wrap already made.

## Handoff

G-0216, G-0286, and G-0334 — the three gaps this epic closes — are each promoted to `addressed`, citing their implementing milestone, satisfying this epic's own success criteria. Nothing is deliberately left open; the epic's Out of scope section (G-0252 red-first TDD ordering; `wf-tdd-cycle` ritual-content changes) stands as written and was not touched.
