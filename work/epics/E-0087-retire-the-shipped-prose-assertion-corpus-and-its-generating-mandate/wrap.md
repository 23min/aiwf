# Epic wrap — E-0087

**Date:** 2026-08-22
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0087-retire-the-shipped-prose-assertion-corpus-and-its-generating-mandate
**Merge commit:** recoverable from `aiwf history E-0087` — the `aiwf-verb: wrap-epic` commit

## Milestones delivered

- M-0312 — Re-point the skill-edit backstop from content reference to provenance (merged 2270b0639)
- M-0313 — Retire the prose-assertion corpus over shipped surfaces (merged a75f0b3ae)

## Summary

The epic set out to remove a body of tests that assert shipped prose still says
particular things, and to re-point the chokepoint that generated them. Both
landed. The backstop now asks a skill edit *who owns it* — an `aiwf-entity`
trailer resolving in the tree — rather than whether some policy test names its
path, which is what turned a per-edit mandate into a rule that costs once. The
corpus is gone: 129 test functions and 18 files, with the policy suite's own
coverage rising rather than falling.

Scope shifted once, deliberately. The deletion was specified over
`internal/policies`; measurement found the same class in `internal/skills`,
reached through `go:embed` rather than a path, and D-0070 scopes itself by
surface rather than by package — so the ban scans every test package and the
milestone spec was amended to match.

The honest headline is what nearly went wrong. The new gate's acceptance
criterion claimed a green suite proved the corpus gone; it did not, and three
independent reviewers found eleven surviving members plus four protected checks
the deletion pass had wrongly taken. That was caught by review, not by the gate —
which is the disposition D-0070 chose for content, arriving sooner than expected
and on the gate itself.

## ADRs ratified

- none

## Decisions captured

- D-0072 — the shipped-prose ban is partial by design and exempts derived expectations

D-0070 and D-0071 govern this epic and were accepted before it started; D-0072
records what implementing them settled that their text did not.

## Follow-ups carried forward

- G-0504 — `aiwf doctor`'s byte-check coverage over ritual and guidance artifacts (named out of scope by the epic spec, unchanged by it)
- G-0605 — investigate type-aware static analysis for aiwf's self-validation (aspirational; analysis in `docs/explorations/10-type-aware-static-analysis.md`)
- G-0606 — a prose assertion written as a production policy escapes the ban
- G-0607 — regexp content matching over shipped surfaces bypasses the ban
- G-0608 — negative regression pins are a class D-0070 does not name
- G-0601 — `aiwf history` hides entity-only-trailered skill edits
- G-0602 — conflict-resolving merges escape the provenance gate
- G-0603 — no early chokepoint for the provenance backstop
- G-0604 — four copies of the id-resolves-in-tree walk

Gaps closed by this epic: G-0596 and G-0584 (both `addressed`); G-0317 resolved
`wontfix`.

## Doc findings

`wf-doc-lint` over the epic's change-set: clean. Four broken markdown links
appear in `ROADMAP.md`, all pre-existing in mainline's copy and reproduced by the
regenerator from entity bodies citing archived paths. Not introduced here and not
fixable in a generated file — the citations belong to the entities.

Three surfaces describing the discipline were reconciled, which D-0070
§Consequences requires: CLAUDE.md's ritual-authoring section now states the
provenance rule, its "Substring assertions are not structural assertions" rule
carves out shipped surfaces (where the answer is to drop the substring, not scope
it), and its AC-evidence section notes that a doc-shaped AC over a shipped surface
cannot use a phrase assertion as evidence.

## Handoff

Ready: the ban is in place across every test package and the retrofit is done.
Adding an exemption now requires choosing one of two named classes, and a case
fitting neither is a signal the class set needs a decision.

Deliberately left open: the three escapes above (G-0606 / G-0607 / G-0608) are
recorded rather than closed, because each needs a disposition rather than a
patch. G-0605 is the larger question behind all three — every syntax-only check
in this repo matches spellings rather than meaning, and whether that is worth
fixing wants evidence a second motivating case would supply.
