# Epic wrap — E-0021

**Date:** 2026-05-09
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0021-recommended-sequence
**Merge commit:** `ca0d601` (epic/E-0021 merged into main with `--no-ff` from the parent worktree at `/Users/peterbru/Projects/ai-workflow-v2`; auto-merge resolved `.gitignore` and `CLAUDE.md` cleanly, with the parent's mid-flight Q&A-tightening edit on `CLAUDE.md` and the AC-mechanical-evidence + cross-repo plugin testing additions both preserved; `STATUS.md` regenerated post-merge to reconcile divergence)

## Milestones delivered

- **M-0078** — Planning-conversation skills design ADR (placement, tiering, name rationale) — merged via patch wrap commit `d447e0f`
- **M-0079** — aiwfx-whiteboard skill: classification rubric, output template, Q&A gate — merged via patch wrap commit `d447e0f`
- **M-0080** — Whiteboard skill fixture validation; retire critical-path.md; close E-0021 — promoted in commit `89791e1`

## Summary

E-0021 graduated the open-work synthesis pattern from a one-off `critical-path.md` snapshot into the reproducible `aiwfx-whiteboard` skill. The skill now ships in the rituals plugin, materialises into the marketplace cache via `aiwf init` / `aiwf update`, and answers natural-language direction questions (*"what should I work on next?"*, *"draw the whiteboard"*) by synthesising tree state into a tiered landscape, recommended sequence, first-decision fork, and Q&A-gated pending decisions. The original holding doc `work/epics/critical-path.md` is retired; the standing `unexpected-tree-file` warning that lived from 2026-05-08 to 2026-05-09 is gone. ADR-0007 captures the placement/tiering rationale (rituals-plugin placement; pure-skill first; kernel verb only when usage demands it). Three follow-up gaps (G-0088, G-0089→addressed, G-0090) carry the loose ends forward.

The epic also surfaced two cross-cutting doctrines that landed in CLAUDE.md mid-flight: the *AC promotion requires mechanical evidence* rule (born from M-0078's "wrapped without tests" episode and the user's *"everything should be tested, not assumed via conversation"* correction) and the *Cross-repo plugin testing* pattern (fixture in this repo as the canonical authoring location during the milestone; deploy step at wrap; drift-check test against the marketplace cache). Both rules are now codified and apply to every future plugin-side milestone.

## ADRs ratified

- *(none — `ADR-0007` was authored under M-0078 and remains at status `proposed` per the spec's explicit deferral; ratification to `accepted` is a separate decision the operator may take post-wrap)*

## Decisions captured

The substantive decisions that surfaced during the epic are codified in their canonical homes rather than as new D-NNN entities:

- **Cross-repo plugin testing pattern** — codified in `CLAUDE.md` §"Cross-repo plugin testing" (commit `31c7b43`). Fixture-first authoring at `internal/policies/testdata/<skill>/SKILL.md`; deploy to rituals repo at wrap; drift-check test against marketplace cache. Subtree/submodule explicitly rejected.
- **AC promotion requires mechanical evidence** — codified in `CLAUDE.md` §"AC promotion requires mechanical evidence" (commit `31c7b43`). Born from M-0078's force-reversal episode; applies even to `tdd: none` milestones. Memory pointer kept at `feedback_ac_mechanical_evidence` per the user's *"memory is not a rule channel"* meta-rule.
- **Whiteboard verb name follows skill name** — captured in ADR-0007 §Tiering (commit `834acf2`). The deferred kernel verb backing `aiwfx-whiteboard` is `aiwf whiteboard`, not `aiwf landscape`. Earlier drafts used `landscape`; user-corrected mid-cycle to keep the surface unified across plugin and kernel.
- **Whiteboard output ordering — action-shaped blocks lead** — captured in patch commit `acf87cc`. Sequence/fork/pending lead the rendered output; tiered landscape moves to last as supporting reference. Reversed the original spec ordering after operator review of the rendered fixture.
- **WHITEBOARD.md gitignored local cache** — captured in patch commits `cbd3021` (kernel-side) and `8f5b946` (rituals-side). Anti-pattern #3 narrowed from "no synthesis snapshot of any kind" to "no checked-in snapshot"; gitignored caches explicitly OK with `STATUS.md` as precedent.

No standalone D-NNN entities filed — the codification surfaces above (CLAUDE.md, ADR-0007, commit messages) are the discoverable record.

## Follow-ups carried forward

- **G-0088** — Kernel skill-coverage policy walks `internal/skills/embedded/` only; plugin skills are not policed by the kernel. Per-skill equivalent invariants must be re-applied in test code as M-0079 did. *Open*; small milestone to expand kernel policy scope.
- **G-0090** — `TestAiwfxWhiteboard_AC8_MaterialisationDriftCheck` has three branches not unit-tested (cache-root absent skip, plugin dir read error skip, fixture/cache drift FAIL); refactor lookup to take cache root as parameter for hermetic testing with synthetic temp dirs. *Open*; wf-patch.

G-0089 (whiteboard gitignored cache + AC-6 anti-pattern revision) was *addressed* by patch `cbd3021`/`8f5b946` mid-epic.

## Doc findings

`wf-doc-lint` scope = files touched on `epic/E-21-recommended-sequence` since diverging from `main`:

- `CLAUDE.md` — added two top-level sections ("AC promotion requires mechanical evidence", "Cross-repo plugin testing"). Both cross-reference existing engineering principles correctly.
- `docs/adr/ADR-0007-*.md` — new ADR; cross-references to ADR-0006, M-0079, E-0021, and CLAUDE.md sections all resolve.
- `internal/policies/testdata/aiwfx-whiteboard/SKILL.md` — fixture for the deployed skill. Cross-references to `aiwf` verbs all resolve (`TestAiwfxWhiteboard_AC7_SkillCoveragePolicyEquivalent` enforces this mechanically).
- `WHITEBOARD.md` — gitignored cache; not part of doc-lint scope.

No broken code references; no removed-feature drift; no orphan docs introduced; no documentation TODOs added. **doc-lint: clean.**

## Handoff

**Ready to start next:**
- `aiwfx-whiteboard` is live in the marketplace cache and routable via natural-language queries. Future operator sessions get direction synthesis on demand.
- Three pending decisions in the live whiteboard output that warrant near-term attention:
  - First-decision fork: A (E-0016 TDD chokepoint), B (E-0019 parallel TDD subagents — needs milestones planned), C (verb-hygiene umbrella from ADR-0005).
  - Should the Tier-1 hygiene sweep (G-0080, G-0083, G-0088, G-0090) be one batched wf-patch or one per gap?
  - ADR ratification questions: ADR-0001 / 0003 / 0004 / 0005 / 0006 / 0007 all sit at `proposed`. Worth a focused ratification pass, possibly bundled.

**Deliberately left open:**
- ADR-0007 ratification (per spec deferral; promote to `accepted` when consensus is ready).
- The deferred `aiwf whiteboard` kernel verb (per ADR-0007's pure-skill-first tiering rule; trigger condition is "skill body re-derives the same structured data on every invocation" — not yet observed).
- G-0088 / G-0090 stay open as filed; trigger their work when the friction shows up or they bundle into a Tier-1 sweep.

**The aiwfx-whiteboard skill itself is the recommended next-step generator.** Re-run the skill against the live tree any time direction synthesis is needed. The deferred-decision ADR ratification fork (above) is where the next epic shape becomes clearer.
