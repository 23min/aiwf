---
id: G-0177
title: 'Rituals distribution: marketplace-only install friction blocks agent-agnostic'
status: addressed
addressed_by:
    - E-0038
---
## Problem

aiwf ships two layers of advisory artifact with sharply different distribution friction:

| Layer | What | How it reaches a consumer repo today | Friction |
|---|---|---|---|
| Verb skills (16) | `aiwf-add`, `aiwf-check`, … one per kernel verb | **Embedded in the binary**, materialized by `aiwf init` / `aiwf update` into `.claude/skills/aiwf-*` (gitignored, marker-managed, auto-refreshed on upgrade) | ~zero |
| Rituals | `aiwfx-*` planning/lifecycle skills, `wf-*` engineering skills, the `planner`/`builder`/`reviewer`/`deployer` agents, templates | **External Claude marketplace plugin** (`23min/ai-workflow-rituals`), installed manually via the interactive `/plugin` menu at *project* scope; **no auto-update**; `aiwf doctor` only *warns* if absent | all of it |

The friction lives entirely in the second layer, and it is structural, not cosmetic:

- **Manual, easy-to-miss install.** The CLI form `claude /plugin install <name>@<marketplace>` defaults to *user* scope; only the interactive `/plugin` menu offers project scope. An operator who clones the repo and runs `aiwf` gets the planning data layer but none of the rituals until they perform a multi-step plugin dance (see CLAUDE.md § "Operator setup").
- **No upgrade path.** Nothing pulls a newer ritual version on `aiwf update`. The plugin version drifts independently of the binary it shells out to.
- **Platform-fragile.** The Claude plugin index stores absolute host paths (anthropics/claude-code#31388), which is why the devcontainer needs a plugin-index shadow-mount. The marketplace channel inherits that fragility.

## Why it matters

Two distinct costs, one of them strategic:

1. **Onboarding and upgrade friction** for every consumer, today. `aiwf` is "just the planning data layer" until the operator completes manual plugin setup, and there is no mechanical refresh.
2. **Hard blocker on the agent-agnostic future.** aiwf is expected to become agent-agnostic — usable from agents other than Claude Code. The Claude marketplace is a Claude-only channel by construction; it can never deliver rituals to Codex, Cursor, Copilot, or any other agent. The distribution mechanism, not the skill *format*, is the obstacle: SKILL.md (`name`/`description` frontmatter) is now a cross-vendor open standard (agentskills.io; OpenAI Codex implements the identical frontmatter, reading skills from `.agents/skills/` rather than `.claude/skills/`). Portability is a question of *output location + delivery channel*, which the marketplace forecloses.

The verb-skill layer already proves the friction-free model: embed in the binary, materialize on `init`/`update`, and the output directory is just a parameter. The rituals layer was deliberately split into a marketplace plugin (ADR-0007, `rituals-plugin-plan.md`) for independent versioning and `wf-*` standalone reuse — but that split is the source of both costs above.

## Proposed direction

Make `aiwf` itself the distribution mechanism for the rituals, the same way it already is for verb skills: **vendor a pinned snapshot of the rituals into the aiwf repo, embed it, and materialize it on `aiwf init` / `aiwf update`** — through a materializer parameterized by agent target so `.claude/skills/` (and `.claude/agents/`) today generalizes to Codex `.agents/skills/` and down-converted Cursor/Copilot rules later. Retire the marketplace channel once the embedded path is stable.

The decision is recorded in an ADR; the work is carried by an epic (both referenced below). The upstream `23min/ai-workflow-rituals` repo stays the authoring home (preserving `wf-*` standalone reuse); aiwf vendors a pinned snapshot from it.

## Out of scope

- **Runtime network fetch of the rituals** at `init`/`update` time. Build-time embed gives version-binding, reproducibility (covered by `go.sum`), and a single self-contained artifact for no extra runtime machinery. A third-party / remote skill registry is explicitly out of scope per CLAUDE.md § "What is *not* in scope".
- **An MCP server exposing the verbs.** A plausible complementary cross-agent track for *verb execution*, but MCP has no primitive that replicates an auto-discovered, progressively-disclosed skill, so it is not the skill-distribution mechanism. Separate concern.
- **The project-prose layer (`CLAUDE.md` vs `AGENTS.md`).** Cross-agent prose portability is a different surface with its own convention. Separate concern.
- **New ritual content.** This is a distribution change, not an authoring one.

## Related

- **ADR-0007** — placement decision being revised. Its *placement/authoring* layering (rituals authored as `aiwfx-*`/`wf-*`, distinct from kernel verb-wrapper skills) is preserved; only its *delivery channel* assumption ("distributed via the Claude Code marketplace") changes.
- **G-0175** — sibling rituals-distribution concern (plugin emits `aiwf-verb:` trailers for non-verb operations).
- **`docs/pocv3/plans/rituals-plugin-plan.md`** — the current marketplace design this supersedes.
- **CLAUDE.md** § "Operator setup", § "Cross-repo plugin testing", and commitments #5 (marker-managed artifacts regenerated on `init`/`update`) and #6 (layered location-of-truth) — the properties this change extends.
