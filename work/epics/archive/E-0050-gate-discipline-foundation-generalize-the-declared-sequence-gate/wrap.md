# Epic wrap — E-0050

**Date:** 2026-06-29
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0050-gate-discipline-foundation-generalize-the-declared-sequence-gate
**Merge commit:** 7e6be1f1

## Milestones delivered

- M-0209 — Generalize the declared-sequence gate; fix wrap/release drift (merged 9644942a)

## Summary

E-0050 generalized the wf-patch-only "declared-sequence gate" into a general
capability for any sequence of *local, reversible* mutations — one gate that
enumerates every action verbatim, is subset-approvable, and aborts + re-gates on
deviation — and fixed the three rituals that violated the gate discipline
CLAUDE.md claimed they followed. The load-bearing addition is the bright line:
batch local/reversible mutations; never batch outward/irreversible actions (push,
`gh pr create`, tag-push, remote-branch delete, `--force`) or timing-bearing ones
(`tdd: required` phase promotes fire live). `aiwfx-release` now splits its two
origin pushes into two gates; `aiwfx-wrap-milestone` and `aiwfx-wrap-epic` run
their terminal local sequence (merge → promote-done → cleanup) under one
declared-sequence gate with push and origin-deletes excluded. Each fix is pinned
by a section-scoped structural test under `internal/policies/`. The seed gap
G-0295 is resolved by this work.

## ADRs ratified

- None new. The gate doctrine landed in CLAUDE.md §"Gate discipline survives
  compaction" (the authoritative governance surface) and the embedded guidance;
  ADR-0024 (filed during planning) covers the related reference-skill mechanism
  for cross-ritual deduplication (E-0048 work, not E-0050).

## Decisions captured

- The declared-sequence gate generalization and its bright line — recorded as
  doctrine in CLAUDE.md and `aiwf-guidance.md`, not as a standalone D-NNNN.

## Follow-ups carried forward

- None opened by E-0050. The commit/TDD model (G-0293) and the opt-in
  declared-sequence-wraps knob (G-0296) remain with E-0049; the wf-tdd-cycle
  audit-before-`met` reorder (G-0309) and the trailered-commit dedup (M-0210) are
  E-0048 work.

## Doc findings

The change-set (CLAUDE.md, embedded guidance, three ritual `SKILL.md`s, two test
files) was reviewed by an independent fresh-context reviewer that verified every
step-number cross-reference and skill link in the changed rituals. No broken
references or removed-feature docs. **Clean.**

## Handoff

The gate model is now coherent across the rituals and matches CLAUDE.md. Both
E-0048 and E-0049 milestone wraps inherit the corrected declared-sequence gate.
The materialized consumer rituals reflect this change after the next `aiwf
update`. No release is bundled with this wrap; cut one via `aiwfx-release` if
desired.
