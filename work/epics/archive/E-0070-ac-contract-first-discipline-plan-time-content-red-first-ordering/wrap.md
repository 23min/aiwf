# Epic wrap — E-0070

**Date:** 2026-07-24
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0070-ac-contract-first-discipline-plan-time-content-red-first-ordering
**Merge commit:** 31dfdbd1

## Milestones delivered

- M-0274 — Seed tdd:required ACs pre-cycle so red is a live event (merged 6002001b)
- M-0275 — Create AC content at plan time; warn on incomplete draft milestones (merged 1b1ad96c)
- M-0276 — Gate red-first ordering via a working-tree diff-shape check (merged bc8c2f86)

## Summary

E-0070 makes the contract-first, red-first AC discipline mechanical rather than
advisory (D-0047). ACs are created and body-filled at plan time, with a
`milestone-draft-incomplete-acs` check catching a draft milestone that reaches
main with zero or empty ACs (M-0275). A new AC seeds at the pre-cycle empty
phase, so the `"" -> red` promote is a genuine live event proving the test came
first (M-0274). An opt-in working-tree diff-shape gate on `--phase red` refuses
when implementation is dirty before the test — proving file-touch ordering
without running tests (M-0276). Mid-wrap the gate was narrowed to **red-only**
(D-0049): the green half false-refused test-only ACs.

## ADRs ratified

- none

## Decisions captured

- D-0047 — Contract-first AC timing and red-first ordering enforcement
- D-0049 — Red-first ordering gate is red-only; drop the green gate

## Follow-ups carried forward

- G-0445 — diff-shape gate hardcodes docs/ exclusion, wrong for some consumer repos

## Handoff

The red-first gate ships opt-in and inactive (no `tdd.test_paths` in this repo's
own `aiwf.yaml`); activating it here to dogfood is a deliberate operator
decision left open. G-0445 (make the excluded path set configurable, or scope it
to `work/` only) is the one design follow-up. The `wf-tdd-cycle` skill now
distinguishes ordering-red from semantic-red for the compile-stub case.
