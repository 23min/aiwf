# Epic wrap — E-0017

**Date:** 2026-05-07
**Closed by:** human/peter
**Integration target:** poc/aiwf-v3
**Epic branch:** epic/E-0017-entity-body-prose-chokepoint-closes-g-058
**Merge commit:** _(filled at step 5)_

## Milestones delivered

- M-0066 — aiwf check finding entity-body-empty (merged 40e192e)
- M-0067 — aiwf add ac --body-file flag for in-verb body scaffolding (merged 7123c74)
- M-0068 — aiwf-add skill names fill-in-body as required next step (merged b075a29)

## Summary

E-0017 closes [G-0058](../../gaps/G-058-ac-body-sections-ship-empty-no-chokepoint-enforces-prose-intent.md) by making non-empty body prose a kernel-enforced property across every entity kind, not just ACs. M-0066 lands the `entity-body-empty` rule (warning by default; error under `aiwf.yaml: tdd.strict: true`) parameterized over kind, with per-kind body-section dispatch and an asymmetric semantics call (top-level sections count sub-headings as content; AC bodies require leaf prose, captured in [D-0001](../../decisions/D-001-entity-body-empty-top-level-sections-count-sub-headings-as-content-only-ac-leaf-bodies-require-non-heading-prose.md)). M-0067 extends `aiwf add ac` with `--body-file` (positional pairing for batched ACs, stdin shorthand for single ACs, leading-`---` rejection across both forms) so AC body content can land in the same atomic create commit as the heading. M-0068 teaches the `aiwf-add` skill to name "fill in the body" as the required follow-up step across all kinds, with per-kind shape recommendations and explicit cross-references to M-0066 and M-0067 — operators reading the skill alone get the full non-empty-body picture from rule to verb to workflow.

The epic also produced one rescope (G-0063 — recorded in the spec's `## Rescope note` section) and surfaced two follow-up gaps that survive the wrap: G-0067 captures the wf-tdd-cycle TDD-discipline slip observed during M-0066/AC-1, and G-0068 captures the discoverability policy's blind spot on dynamically-derived finding subcodes.

## ADRs ratified

- _(none — no architectural choices warranted ADR scope; the asymmetric-semantics call landed as D-0001 instead, which is the right granularity for that decision)_

## Decisions captured

- D-0001 — top-level sections count sub-headings as content; AC bodies require non-heading prose (M-0066/AC-1 wrap-time)

## Follow-ups carried forward

- G-0067 — wf-tdd-cycle is LLM-honor-system advisory under load (M-0066/AC-1 process retrospective)
- G-0068 — discoverability policy misses dynamic finding subcodes (M-0066/AC-6 sanity-check discovery)
- G-0066 — closed `addressed --by M-056,M-067` on this branch immediately before wrap; surfaces here only because the closure rides this epic's merge into mainline

## Doc findings

`wf-doc-lint` scoped to E-0017 branch since `poc/aiwf-v3`:

- **Broken code references:** none.
- **Removed-feature docs:** none. E-0017 is purely additive (new rule, new flag, new skill content, no deletions).
- **Orphan files:** none added under `docs/`.
- **Documentation TODOs:** none introduced. (Pre-existing `TODO`-as-content mentions in `docs/pocv3/migration/from-prior-systems.md` are about TODO logs as a migration concept, not action items, and are not from this epic.)

E-0017's narrative-doc footprint is empty — the epic touched zero files under `docs/`. The new symbols (`entity-body-empty`, `tdd.strict`, `--body-file`, the `aiwf-add` skill content) reach AI-discoverable channels through embedded SKILL.md edits, structurally enforced by `PolicyFindingCodesAreDiscoverable` and the new `TestSkill_*` content tests.

## Handoff

The chokepoint is now mechanical: `aiwf check` flags empty load-bearing body sections at warning severity by default and at error severity under `tdd.strict: true`; the `aiwf-add` skill teaches operators (and LLMs) to fill them in by default rather than skip; `--body-file` removes the friction excuse for both top-level kinds (M-0056, pre-E-0017) and ACs (M-0067, this epic). G-0058's "no chokepoint enforces prose intent" original observation no longer holds — the chokepoint exists, the verb supports the workflow, and the skill teaches it.

What's deliberately left open: backfill of historical bodies (called out as out-of-scope in the epic spec; M-0049..M-0061 surface as warnings under the new rule per their original empty-body shape, with M-0066/AC-1's backfill having taken care of the kernel repo's 62 findings via stub-prose entries plus M-0061's real `## Goal`), and the discoverability policy gap (G-0068) which is its own kernel-discipline concern. Either can become a milestone in a future epic; neither blocks E-0017's claim that "non-empty body prose is now a kernel-enforced property."
