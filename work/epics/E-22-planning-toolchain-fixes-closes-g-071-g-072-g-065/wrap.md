# Epic wrap — E-22

**Date:** 2026-05-08
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-22-planning-toolchain-fixes-closes-g-071-g-072-g-065
**Merge commit:** `4ebb939`

## Milestones delivered

- M-075 — Lifecycle-gate `entity-body-empty` rule (closes G-071) (merged `124c9cb`)
- M-076 — Writer surface for milestone `depends_on` (closes G-072) (merged `42cbafb`)
- M-077 — `aiwf retitle` verb for entities and ACs (closes G-065) (merged `0f8c8a0`)

## Summary

E-22 was the pre-E-20 cleanup pass: three Tier-1 planning-toolchain frictions whose cumulative tax on every multi-milestone planning session justified bundling them. M-075 lifecycle-gated the `entity-body-empty` rule, dropping the kernel repo's warning baseline from 46 to ~1 — pre-implementation drafts and post-terminal artifacts no longer ship perpetual noise. M-076 closed the kernel asymmetry where `depends_on` had six read sites and zero writers; the new `--depends-on` flag on `aiwf add milestone` and the dedicated `aiwf milestone depends-on M-NNN --on … [--clear]` verb both produce trailered atomic commits, with referent validation and forward-compatibility with G-073's eventual cross-kind generalisation. M-077 added the `aiwf retitle` verb (top-level kinds + composite-id ACs) so scope refactors can correct titles without leaving frontmatter `title:` permanently misleading; ships with a dedicated skill plus a redirect from `aiwf-rename`.

After E-22, planning a multi-milestone epic produces a clean tree, milestones declare their DAG via verb, and titles can be corrected when scope shifts. The three target gaps (G-071, G-072, G-065) are closed by their wrap commits.

## ADRs ratified

- (none — every locked design decision in this epic was pre-locked in the epic spec or the per-milestone spec; nothing surfaced mid-implementation that warranted ADR shape)

## Decisions captured

- (none — no D-NNN entities filed during the epic)

## Follow-ups carried forward

- G-079 — `aiwfx-plan-milestones` plugin skill needs `--depends-on` documentation; M-076 added the verb but the plugin lives in the `ai-workflow-rituals` repo upstream. Filed by M-076 wrap; closes when the plugin update lands.
- G-073 — broader cross-kind `depends_on` generalisation (schema relaxation, per-kind `SatisfiesDependency(kind, status)` predicate, status-aware FSM gating, reverse query). Open as the design lens; awaits its own implementation epic when E-21's synthesis skill or another consumer pays for the prose-only fallback.
- G-059 — branch model: no canonical mapping from entity hierarchy to git branches. Surfaced when the user noted M-075 was wrapped on `main` rather than on `epic/E-22-...`/`milestone/M-075-...`. Prioritized to land before the next epic per user direction.
- G-063 — no defined `start-epic` ritual: epic activation is a sovereign act with preflight + optional delegation, but the kernel treats it as a one-line FSM flip. Companion to G-059; same priority window.

## Handoff

**Ready for the next epic:**
- The new writer verbs (`aiwf milestone depends-on`, `aiwf retitle`, `--depends-on` flag) are AI-discoverable through the embedded skills (`aiwf-add`, `aiwf-retitle`) and the README verb table.
- The `entity.IsTerminal(kind, status)` helper added by M-075 is available to E-20/M-072 if it ships next (the helper was originally coordinated as a shared addition; M-075 added it first).
- The kernel repo's `aiwf check` baseline is clean apart from the standing `unexpected-tree-file` on `work/epics/critical-path.md` (E-21's scope) and `provenance-untrailered-scope-undefined` warnings on branches without an upstream.

**Deliberately left open:**
- Cross-kind `depends_on` (G-073), reverse query, status-aware FSM gating — all G-073 territory; M-076's verb shape is a clean subset of the future generalisation, so when that epic ships, today's verbs extend without rename.
- Phase gating on the `entity-body-empty` rule (`tdd_phase`-aware AC skipping) — M-075 deferred this since status gating already covers both G-071 cases; revisit if precision-need surfaces.
- Per the user's direction, G-059 and G-063 are prioritised before the next planning epic.

## Doc findings

doc-lint: clean. No broken code references, no removed-feature docs, no orphan files surface in the epic's diff. README.md was updated during M-077 wrap to add the new verbs (`aiwf retitle`, `aiwf milestone depends-on`) and refresh the materialized-skills count from ten to twelve. The `depends_on` mentions in `docs/research/` and `docs/archive/` describe structural properties (cycle detection, closed reference set, conceptual edge types) rather than specific verb shapes; M-076's actual verb shape doesn't contradict the research framing, and the archive content is preserved as historical context per the design.

The kernel's two unrelated standing warnings remain:
- `unexpected-tree-file: work/epics/critical-path.md` — E-21's scope (the synthesis-skill epic), not E-22's.
- `provenance-untrailered-scope-undefined` — surfaces only on branches without a configured upstream; goes silent once the branch is pushed.
