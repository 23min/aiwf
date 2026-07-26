# Epic wrap — E-0072

**Date:** 2026-07-26
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0072-cli-verb-scaffold-convergence
**Merge commit:** ccf877b0

## Milestones delivered

- M-0278 — Extract BeginVerbDiag and migrate the verb diagnostic block (merged 281b5897)
- M-0279 — Extract the shared ResolveRoot to ResolveActor prelude (merged fb873b04)
- M-0280 — Regression-guard structural test for the verb scaffolds (merged 87300fd2)

## Summary

Single-sourced the two copy-pasted per-verb scaffolds in `internal/cli`: the diagnostic-logging block (now `cliutil.BeginVerbDiag` / `BeginReadVerbDiag`) and the `ResolveRoot → ResolveActor` prelude (now `cliutil.ResolvePrelude` / `ResolvePreludeEnvelope`), migrating every verb that hand-rolled them. Both migrations are behavior-preserving — the emitted diagnostic events and `--format=json` envelopes are byte-identical to pre-migration, verified by a runtime binary diff across the migrated verbs. The convergence is then locked by a structural regression-guard policy (`PolicyVerbScaffoldSingleSeam`, `internal/policies`) that fails if any verb re-inlines either scaffold, with a relocation anchor so a future `cliutil` split fails loud rather than rotting the guard green. This completes the remaining scope of G-0447.

## ADRs ratified

- none — behavior-preserving refactor following existing patterns; design rationale is captured in each milestone spec's Reviewer notes.

## Decisions captured

- none as ADR/D entities — the design forks (two-helper prelude, single-primitive keying plus the relocation anchor, and extracting into today's `cliutil` in place rather than pre-accommodating the G-0227 split) were resolved by independent review and recorded in the milestone specs' Reviewer notes.

## Follow-ups carried forward

- G-0456 — prelude resolution error format is not uniform across verbs (filed in M-0279; a pre-existing inconsistency preserved by the behavior-preserving refactor).
- G-0453 — SHA-abbreviation width decision (sub-seam split from G-0447 during planning; out of scope here).
- G-0454 — id-parser assumes-kind vs discovers-kind (sub-seam split from G-0447; out of scope here).
- G-0455 — body.go heading-walker consolidation (sub-seam split from G-0447; out of scope here).

## Doc findings

`wf-doc-lint` scoped to the epic change-set: clean. The epic touched no files under `docs/`, `README`, `CHANGELOG`, or `CONTRIBUTING` — the change-set is `internal/` source, `internal/policies` tests, `ROADMAP.md`, and the epic's own entity files. `aiwf check` reports 0 error-severity findings.

## Handoff

The two per-verb scaffolds are single-sourced and mechanically guarded. The codebase is ready for the G-0227 `cliutil` split: the guard's relocation anchor is designed to fail loud if that split relocates the wrapped primitives, forcing the guard's package key to be updated alongside the move rather than silently rotting green. G-0453 / G-0454 / G-0455 (the decision/risk sub-seams split from G-0447) and G-0456 remain open for future work.
