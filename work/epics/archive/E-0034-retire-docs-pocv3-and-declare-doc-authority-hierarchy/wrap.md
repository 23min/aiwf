# Epic wrap — E-0034

**Date:** 2026-07-21
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0034-retire-docs-pocv3-and-declare-doc-authority-hierarchy
**Merge commit:** 85734b9a

## Milestones delivered

- M-0126 — Triage docs/pocv3/ into per-file disposition table (merged de49381b)
- M-0127 — Relocate docs/pocv3/ contents and sweep cross-references (merged 4eecf22b)
- M-0128 — Declare doc-authority hierarchy in CLAUDE.md (merged f7f72d29)
- M-0129 — Drift chokepoint: forbid docs/pocv3/ literals in Go code — cancelled (superseded before start by M-0127's existing mechanical guard)

## Summary

E-0034 retired `docs/pocv3/`, the working-name vintage from before the trunk promotion, and declared a documented authority hierarchy over the rest of `docs/`. M-0126 produced `TRIAGE.md`, a 42-row per-file disposition table (relocate / archive / supersede-with-entity) covering every file under the old tree. M-0127 executed that table verbatim — 42 physical moves, a repo-wide cross-reference sweep (~121 files: Go source, embedded skills, docs, ADRs, CI config, planning-tree bodies), and a new permanent mechanical assertion (`TestM0127_AC3_NoDanglingDocsPocv3References`) that the literal substring `docs/pocv3` appears nowhere live outside a narrow, rationale-carrying allowlist. `docs/pocv3/` no longer exists. M-0128 added a `## Documentation hierarchy` section to CLAUDE.md naming every active `docs/` subtree and top-level narrative file under one of four closed-set tiers (normative / forward-looking / exploratory / archival), backed by a structural test. M-0129, the epic's originally-planned drift chokepoint, turned out to be redundant before implementation started: M-0127's own AC-3 test already provides the exact guarantee M-0129 proposed, at a broader scope (Go and markdown, not Go alone) — it was cancelled with rationale rather than shipping a duplicate check. The epic's own success criteria also named three open gaps (G-0074, G-0075, G-0092) whose entire subject was `docs/pocv3`'s retirement or the doc-hierarchy question; all three are now superseded by E-0034 and archived as part of this wrap.

## ADRs ratified

- none

## Decisions captured

- none — two milestone-scoped judgment calls (M-0127's `PolicyDesignDocAnchors` scope narrowed to `docs/design/` rather than broadened to all of `docs/`; M-0128's tier assignment for the two ambiguous subtrees, `docs/research/` and `docs/migration/`) were each explicitly assessed against the ADR/`D-NNN` bar and judged not to clear it — recorded as direct exposition in their own milestone specs (M-0127's AC-4 Work log / Reviewer notes; M-0128's Decisions-made section) rather than as standalone decision records. M-0129's cancellation rationale is preserved in its own `aiwf cancel --reason` commit and `aiwf history M-0129`, the same treatment.

## Follow-ups carried forward

- G-0436 — `CLAUDE.md` and `docs/design/id-allocation.md` cite stale `cmd/aiwf/` paths for two source files an unrelated, pre-existing `cmd/aiwf/` -> `internal/cli/` refactor moved. Surfaced by this wrap's epic-level doc-lint sweep; confirmed pre-existing (byte-identical on `main` before this epic) and orthogonal to E-0034's scope, so filed rather than fixed inline.

Every gap the epic's own success criteria named (G-0074, G-0075, G-0092) is addressed and archived as part of this wrap; G-0436 above is the sweep's one genuine new finding.

## Doc findings

Epic-level doc-lint sweep (independent, fresh-context pass) over the full change-set (`main...epic/E-0034-...`, 63 of 90 changed markdown files linted — 26 under `docs/archive/**` excluded per ADR-0004's forget-by-default convention, 1 file a confirmed-intentional delete-and-merge). Used `lychee` (236 links checked, fragment/anchor resolution against real headings) plus a hand-rolled heading-hierarchy checker and the CLI-help oracle for invocation resolution.

**0 blocking findings.** Two categories of real, epic-caused findings, both fixed in this wrap:

- **3 orphaned files** — `docs/migration/from-prior-systems.md` and both `docs/explorations/loom/` files lost their only inbound link when M-0127 archived `docs/pocv3/README.md`. Fixed: CLAUDE.md's Documentation hierarchy section now links all three directly.
- **1 pre-existing broken relative link the sweep's own path-rewrite touched without fixing** — `work/gaps/G-0168-...md` linked `docs/design/design-decisions.md` with a path relative to repo root instead of `work/gaps/`. Fixed: corrected to `../../docs/design/...`.

One completeness gap in M-0128's own deliverable, also fixed: CLAUDE.md's Normative tier didn't carve out `docs/adr/archive/` under the same forget-by-default convention as the Archival tier — added a one-clause note.

Everything else the sweep found is pre-existing and orthogonal to this epic (2 stale `cmd/aiwf/` citations, now G-0436; 3 pre-existing orphan docs never linked from `docs/pocv3/README.md` either; `docs/archive/README.md` not yet indexing `docs/archive/pocv3/`, already recorded as a deliberate M-0126 decision) — left as-is, consistent with M-0127's own three-reviewer wrap review and its permanent `TestM0127_AC3_NoDanglingDocsPocv3References` regression guard.

## Handoff

`docs/` now carries a documented, mechanically-partially-enforced authority hierarchy: CLAUDE.md's `## Documentation hierarchy` section is the human-facing map, and `TestM0127_AC3_NoDanglingDocsPocv3References` is the permanent regression guard against `docs/pocv3` literals reappearing in Go source, markdown, or the planning tree. Deliberately left open: CLAUDE.md's hierarchy section is a fixed snapshot (no live drift-check against `docs/`'s actual runtime layout), and G-0092's original layer-3 proposal (a kernel rule that checks normative-tree files for stale code/entity references) remains unimplemented — both tracked as future work only if real drift friction shows up, not as a residual gap from this epic.
