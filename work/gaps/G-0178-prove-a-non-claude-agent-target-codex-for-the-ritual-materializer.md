---
id: G-0178
title: Prove a non-Claude agent target (Codex) for the ritual materializer
status: open
---
## Problem

E-0038 introduces an agent-target seam in the ritual materializer (M-0151) but deliberately ships only the Claude writer. The agent-agnostic claim is therefore *structurally* unblocked but not *demonstrated*: no non-Claude target materializes the rituals end-to-end, so the seam is a single-implementation abstraction whose fitness is unproven.

## Why it matters

An abstraction with one implementation is a guess. Per CLAUDE.md KISS/YAGNI, the seam was extracted on the second case coming into view — but until a real second writer exercises it, we don't know the seam carved the right joint. Proving one non-Claude target validates the whole agent-agnostic direction at low cost.

## Proposed direction

After M-0151 lands the seam, implement an **OpenAI Codex** target writer as the first proof: SKILL.md is a cross-vendor open standard, and Codex reads the identical `name`/`description` frontmatter from `.agents/skills/`, so the Codex writer is near-verbatim (output location differs, format does not). Ship it with a fixture test asserting `aiwf init --agent codex` (or the chosen surface) writes `.agents/skills/{aiwfx,wf}-*/SKILL.md`. Agents (`planner`/`builder`/`reviewer`/`deployer`) either map to a Codex equivalent or no-op for that target.

## Out of scope

- **Cursor / Copilot / Windsurf down-conversion writers** — those need skill→flat-rule translation (`.cursor/rules/*.mdc`, `.github/instructions/*.md`); separate follow-ons once the Codex proof validates the seam.
- **The MCP-for-verbs and AGENTS.md-prose tracks** — separate cross-agent surfaces, out of scope per G-0177.

## Related

- **E-0038** — the epic this is deferred from (its optional M6).
- **M-0151** — the agent-target seam this builds on.
- **ADR-0014** §4 (agent-target abstraction).
- **G-0177** — the originating friction + agent-agnostic gap.
