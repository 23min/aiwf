# Epic wrap — E-0043

**Date:** 2026-06-24
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0043-optional-area-tag-for-grouping-entities-by-workstream
**Merge commit:** e3168394

## Milestones delivered

- M-0171 — Area field on root kinds and aiwf.yaml areas block with validation (merged 57e10960)
- M-0172 — area-unknown check finding for undeclared area values (merged 98db0f00)
- M-0173 — aiwf add --area write path with completion and discovered-in derivation (merged 9afbdafa)
- M-0174 — --area filter on list, show, and status (merged 2dc41e7c)
- M-0175 — Area grouping in status, render roadmap, and render html (merged a50e2166)

## Summary

Implements G-0266's converged design: one repo can now hold more than one workstream via an optional, validated `area` tag on the five root kinds (epic, ADR, gap, decision, contract); milestones and ACs derive their area from the parent epic rather than storing it. An `aiwf.yaml: areas` block declares the closed member set plus an optional `default:` label for the untagged complement. The field is inert with no block (zero migration — today's trees validate and render unchanged); once declared, the `area-unknown` check flags undeclared values, `aiwf add --area` writes/validates/tab-completes them (a gap derives its area from `discovered_in`), the read verbs filter by `--area`, and `status` / `render roadmap` / `render --format=html` group by area. The flat, globally-unique id space is untouched (commitment #2) — `area` is a grouping tag, never a directory or id axis.

## ADRs ratified

- none — the architectural decision (the area-tag design) predates this epic in G-0266's "Direction (converged)" section, which the epic implemented verbatim. The rejected alternative (per-area id namespacing) is recorded in the epic spec's *Out of scope*.

## Decisions captured

Recorded inline in the milestone specs (no separate decision entity warranted; the reasoning lives next to the work and in the epic's resolved open questions):

- M-0173 — a gap derives `area` from `discovered_in` on omit; explicit `--area` wins (epic Open Question 1).
- M-0174 — `show --area` is a single-entity predicate; `status --area` scopes the entity sections only (recent/warnings/health stay global); an undeclared `--area` notes to stderr and yields empty (reads never reject — the asymmetry with the `add --area` write path).
- M-0175 — empty *declared* areas are suppressed, the untagged complement is always shown (epic Open Question 2); the grouping target is the epic sections across all three surfaces; AC-4 is asserted by DOM containment (the codebase carries no HTML-parse dependency).

## Follow-ups carried forward

- none filed by this epic. G-0277 (filed on `main` mid-epic — default `aiwf status` shows stale milestone status vs an unmerged epic worktree) is independent of this feature; this merge reconciles it into the epic history, and it remains open for its own resolution.

## Doc findings

`wf-doc-lint` (scoped to the epic change-set): clean. Skill `aiwf <verb>` mentions resolve (the `skill_coverage` policy is green); entity references validate (`body-prose-id` green); the new `--area` flag and area-grouping ship with `--help` text and skill docs (`aiwf-list`, `aiwf-show`, `aiwf-status`, `aiwf-render`, `aiwf-check`). The one relative link a naive scan flagged in `aiwf-render` (governance-html-plan.md) resolves correctly from the materialized `.claude/skills/` path and predates the epic.

## Handoff

The `area` feature is complete and on `main`. A consumer opts in by adding an `areas` block to `aiwf.yaml`; with no block, every surface behaves exactly as before. G-0266 is promoted to `addressed` at this wrap. No release is cut here (that is `aiwfx-release`); the user-visible delta is recorded under `## [Unreleased]` in `CHANGELOG.md`. The terminal entities from this epic (E-0043 and its milestones, addressed G-0266) are eligible for the next `aiwf archive --apply` sweep per ADR-0004 — a separate, tree-wide operation, not bundled into this wrap.
