# Epic wrap — E-0038

**Date:** 2026-05-29
**Closed by:** human/peter
**Integration target:** main
**Epic branch:** epic/E-0038-rituals-distribution
**Merge commit:** 19829699

## Milestones delivered

- M-0148 — Vendor-sync: pull pinned rituals snapshot into the aiwf repo + drift test (merged 2e23bc7c)
- M-0149 — Embed + materialize ritual skills (aiwfx-/wf-); extend manifest + gitignore (merged 3047c662)
- M-0150 — Embed + materialize ritual agents (.claude/agents/) and templates (merged 5ad30cc4)
- M-0151 — Agent-target seam in the materializer (Claude writer behind the seam) (merged ff6b1f1f)
- M-0152 — Marketplace sunset: doctor flip, de-dupe guard, docs rewrite (merged 84f772c9)

## Summary

Delivered the epic goal: the rituals (planning/lifecycle `aiwfx-*` skills, engineering `wf-*` skills, the four role agents, and entity templates) now ship **embedded in the engine binary** from a pinned upstream snapshot and materialize into `.claude/` via `aiwf init` / `aiwf update` — the same marker-managed, gitignored pipeline that already shipped the verb skills. The Claude marketplace channel is retired: `aiwf doctor` verifies the materialized artifacts and runs a de-dupe guard instead of recommending a plugin, and the operator-setup docs are rewritten to the one-command flow. The materializer is parameterized by an agent target (M-0151), so a non-Claude target (Codex, Cursor) becomes a new writer rather than a rewrite — structurally unblocking the agent-agnostic future without building those writers here.

## ADRs ratified

- ADR-0014 — Embed-and-materialize rituals distribution; retire Claude marketplace (proposed → accepted at wrap; the epic proved the design out)

## Decisions captured

- D-0015 — Ritual templates materialize to `.claude/templates/` (Claude target)
- D-0016 — Retire `doctor.recommended_plugins`; verify materialized rituals + de-dupe guard

## Follow-ups carried forward

- G-0178 — Prove a non-Claude agent target (Codex) for the ritual materializer (the M-0151 seam unblocks it; no production writer ships in this epic)
- G-0179 — Enforce the full local CI gate (golangci-lint) at wrap on unpushed branches (surfaced repeatedly during this epic — latent lint that only the local full gate caught, since CI never ran on the unpushed branch)

## Gaps closed

- G-0177 — Rituals distribution: marketplace-only install friction blocks agent-agnostic (addressed by E-0038)

## Handoff

The embed-and-materialize path is complete and CI-clean (`golangci-lint run ./...` 0 issues, `aiwf doctor --self-check` 30 steps, full suite green modulo a known heavy-parallel git-contention flake in `internal/check`). The Claude target is the only shipped writer; the seam is ready for the first non-Claude writer (G-0178). Operators on an existing marketplace install are guided off it by the `marketplace-rituals-overlap` doctor guard (detect-and-instruct; aiwf never edits `settings.json`).

Deliberately left open: the non-Claude target writers (G-0178) and the mechanical local-gate enforcement (G-0179). No release is cut by this wrap — `aiwfx-release` owns version tagging.