# Epic wrap — E-0067

**Date:** 2026-07-18
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0067-harden-the-cross-branch-read-path-end-the-list-check-hang-fix-show-area
**Merge commit:** `d95a9847`

## Milestones delivered

- M-0265 — Make the cross-branch collision scan lazy via a single trunk helper (merged `2a31fa2e`)
- M-0266 — Honor a cross-branch id's real area in show --area (merged `ece3a369`)

## Summary

E-0060 shipped the cross-branch read path but composed its collision scan eagerly at
three call sites, running a `git cat-file` blob-stat per id on every ref — O(entities ×
refs) work whose result is consumed only after a local-tree miss, so in the common
all-merged state nearly all of it was computed and discarded. M-0265 replaced that with a
single `internal/trunk.ScanCrossBranch` helper that runs collision detection only for ids
absent from the local working tree — behavior-preserving by the miss-guard subset
invariant — and consolidated the three call sites into one. M-0266 fixed the one
cross-branch correctness bug living in the same code: `aiwf show <cross-branch-id> --area
X` now evaluates against the resolving ref's real `area:` for the kinds that carry their
own area, instead of reporting the id untagged. Cross-branch rows and check findings are
byte-identical to before; filtered `aiwf list` and `aiwf check` shed the discarded scan.

## ADRs ratified

- ADR-0035 — Cross-branch collision detection is scoped to the locally-absent id set

## Decisions captured

- none — D-0036 (collision severity is non-blocking) pre-existed and is unchanged.

## Doc findings

- Clean. The doc-lint sweep over the epic's change-set found no broken references or
  removed-feature docs.

## Follow-ups carried forward

- G-0421 — cross-branch **milestone** `--area` should honor the parent epic's rolled-up
  area (deferred from M-0266; the own-field read here reports a cross-branch milestone
  untagged).
- G-0416 — distinguish an unmerged edit from a genuine duplicate-mint collision (deferred;
  `ScanCrossBranch` is the seam that makes it a cheap successor).

## Handoff

The cross-branch read path is now fast and consolidated: every consumer composes it
through `trunk.ScanCrossBranch`, and — per ADR-0035's standing obligation — any new reader
of the collision set must guard on a local-tree miss first, or the lazy scoping silently
returns an empty result. The remaining `aiwf check` cost is its full-history revwalk
(G-0372), a separate cost center untouched here. G-0421 (milestone `--area` roll-up) and
G-0416 (unmerged-edit vs duplicate-mint) are the natural next steps on this surface. Two
terminal gaps closed by this epic (G-0418, G-0419) and the epic's own entities await an
`aiwf archive --apply` sweep whenever the tree is next tidied — purely advisory.
