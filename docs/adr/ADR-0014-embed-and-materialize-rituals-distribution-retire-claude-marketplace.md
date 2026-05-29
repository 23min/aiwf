---
id: ADR-0014
title: Embed-and-materialize rituals distribution; retire Claude marketplace
status: accepted
---
## Context

aiwf distributes two layers of advisory artifact, and they reach a consumer repo by two very different mechanisms (G-0177):

- **Verb skills** (`aiwf-*`, one per kernel verb) are **embedded in the engine binary** and materialized into `.claude/skills/aiwf-*` by `aiwf init` / `aiwf update` — gitignored, marker-managed, auto-refreshed on upgrade. Friction is essentially zero.
- **Rituals** (`aiwfx-*` planning/lifecycle skills, `wf-*` engineering skills, the `planner`/`builder`/`reviewer`/`deployer` agents, and templates) are distributed as an **external Claude marketplace plugin** (`23min/ai-workflow-rituals`), installed manually via the interactive `/plugin` menu at project scope, with no auto-update. ADR-0007 and `docs/pocv3/plans/rituals-plugin-plan.md` chose this split for independent versioning and `wf-*` standalone reuse.

Two forces make the marketplace channel the wrong long-term mechanism:

1. **Friction.** Manual, easy-to-miss install (the CLI install form defaults to *user* scope; only the interactive menu offers project scope); no `aiwf update` refresh; and platform fragility from the Claude plugin index storing absolute host paths (anthropics/claude-code#31388, the reason the devcontainer needs a plugin-index shadow-mount).
2. **Agent-agnosticism.** aiwf is expected to become usable from agents other than Claude Code. The Claude marketplace is a Claude-only channel by construction — it can never deliver rituals to OpenAI Codex, Cursor, Copilot, or any other agent. Crucially, the **skill *format* is already portable**: SKILL.md (`name`/`description` frontmatter) is a cross-vendor open standard (agentskills.io; Codex implements the identical frontmatter, reading skills from `.agents/skills/` instead of `.claude/skills/`). The obstacle is the *delivery channel and output location*, not the file.

The verb-skill layer already demonstrates the friction-free, format-portable model: embed once, materialize on `init`/`update`, and the output directory is just a parameter. The decision below extends that model to the rituals.

Two CLAUDE.md commitments bear directly and are *extended* (not contradicted) by this decision:

- **#5 — marker-managed framework artifacts regenerated only on explicit `init`/`update`.** Today scoped to `.claude/skills/aiwf-*` and the git hooks; this decision widens the managed set to the ritual skills, agents, and templates.
- **#6 — layered location-of-truth** (engine external, policy/state in the consumer repo, materialized adapters gitignored). The rituals move from "machine-local plugin cache" to "materialized adapter in the consumer repo," staying on the right side of the layering.

Per CLAUDE.md § "What is *not* in scope", a third-party / remote skill registry and a module system are explicitly excluded — which rules *out* runtime fetch from a remote registry and rules *in* build-time embedding.

## Decision

### 1. Distribution mechanism — build-time embed, materialized on `init`/`update`

The rituals are **embedded in the engine binary via `go:embed`** and written into the consumer repo by `aiwf init` / `aiwf update`, using the same marker-managed, gitignored, wipe-and-rewrite pipeline that already ships the verb skills. There is **no runtime network fetch**. Build-time embed binds the ritual version to the binary version, is reproducible and integrity-checked through the Go module system (`go.sum`), produces a single self-contained artifact, and adds no runtime machinery. The only property runtime fetch would add — updating rituals without reinstalling the binary — is something the design deliberately does not want (it reintroduces a binary-version × ritual-version compatibility matrix) and is foreclosed by version pinning anyway.

### 2. Source of truth — upstream repo authors, aiwf vendors a pinned snapshot

The `23min/ai-workflow-rituals` repo stays the **authoring home**, preserving the `wf-*` skills' standalone reusability and the clean coupling boundary ADR-0007 established. aiwf vendors a **pinned snapshot** of the rituals into a path inside the aiwf repo (e.g. `rituals/`), which `go:embed` then bakes in. The snapshot is recorded as a pinned upstream commit SHA — better provenance than "whatever version the operator happened to `/plugin install`."

The snapshot **must be real committed files, not a git submodule.** `go install …@version` fetches modules through the Go module proxy, which does **not** fetch submodule contents — a submodule would embed as an empty directory and ship a binary with no rituals. The vendoring is a `git subtree`-style or scripted copy committed into the aiwf repo, guarded by a drift-check test, mirroring the existing cross-repo SKILL.md fixture discipline (CLAUDE.md § "Cross-repo plugin testing").

### 3. Artifact coverage — skills, agents, and templates; hooks are out

Rituals comprise three materializable artifact kinds, and **agents are treated exactly like skills**:

- ritual skills → `.claude/skills/{aiwfx,wf}-*/SKILL.md`
- agents → `.claude/agents/*.md`
- templates → their referenced locations

The `.aiwf-owned` manifest and the `.gitignore` patterns extend to own the new `aiwfx-*`/`wf-*` skill dirs and the agents dir, preserving the "never clobber user-authored skills" guarantee.

**Hooks are not part of the rituals.** The `aiwf-extensions` and `wf-rituals` plugins ship skills + agents + templates only — no `hooks/` or `hooks.json` (`rituals-plugin-plan.md`). aiwf's own git hooks (`# aiwf:pre-push`, `pre-commit`, `post-commit`) are already managed by `init`/`update` and are orthogonal. So this decision introduces no new hook surface.

### 4. Agent-target abstraction

The materializer is parameterized by **agent target** from the outset. The Claude target is the only one implemented initially (writing `.claude/skills/` and `.claude/agents/`), but the seam is designed so additional targets are *new writers, not a refactor*:

- Codex: verbatim SKILL.md to `.agents/skills/` (same open standard).
- Cursor / Copilot / others: down-convert skills to that agent's flat rule format (`.cursor/rules/*.mdc`, `.github/instructions/*.md`).
- Agents are the least-portable artifact; for a target with no subagent concept, the agent writer is a no-op for that target.

This mirrors how `doctor.recommended_plugins` is already config-driven rather than hardcoded.

### 5. Marketplace retirement — phased, with a de-dupe guard

The marketplace channel is sunset *after* the embedded path is stable, in phases: (a) ship embed+materialize alongside the still-working marketplace; (b) flip `aiwf doctor` from "recommend the plugin" to "verify the materialized artifacts," and drop `doctor.recommended_plugins` from the default `aiwf.yaml`; (c) deprecate the marketplace and rewrite the operator-setup docs to the one-command flow.

The one hazard is the overlap window: a consumer with *both* the marketplace plugin enabled *and* the materialized `.claude/skills/aiwfx-*` would expose two skills with the same `name:`. The materializer detects an enabled plugin (it already reads `.claude/settings.json` `enabledPlugins`, exactly as `doctor` does today) and **instructs the operator to disable the plugin** rather than silently editing their `settings.json` — quiet mutation of user settings is more invasive than the "regenerate only marker-managed artifacts" posture allows.

### Relationship to ADR-0007

ADR-0007's *placement/authoring* layering — rituals authored as `aiwfx-*`/`wf-*`, distinct from kernel verb-wrapper skills, pure-skill-first — is **preserved**. This ADR revises only its *delivery channel* assumption ("distributed via the Claude Code marketplace"): the same rituals now reach the consumer by embed+materialize instead of marketplace install.

### Reversal

What undoes this? Re-publishing the rituals as a marketplace plugin and removing the embed+materialize path is a same-shaped inverse (another release of the binary plus a `/plugin` install). The vendored snapshot is reversible by un-pinning. No one-way door.

## Consequences

- **One install path and one upgrade path.** `aiwf init` installs verb skills *and* rituals; `aiwf upgrade` (which already re-execs `go install` then `aiwf update`) refreshes both. No `/plugin` dance, no marketplace, no plugin-index path bug.
- **Ritual version ≡ binary version, always.** No compatibility matrix between a floating ritual version and the binary it shells out to. Provenance is a pinned upstream SHA in git history.
- **Binary grows by a few hundred KB of markdown** — negligible next to the Go binary; the 16 verb skills already embed without notice.
- **Independent plugin versioning is given up.** Accepted: a pinned vendor SHA is stronger provenance than marketplace-install drift, and the rituals do not churn faster than the binary.
- **New materialize surface for agents and templates.** Agents are the least portable; the agent-target seam absorbs that without reshaping the pipeline.
- **`rituals-plugin-plan.md`'s marketplace design is superseded** and is updated/archived by the implementing epic; CLAUDE.md § "Operator setup" is rewritten to the one-command flow.
- **The agent-agnostic future is unblocked structurally** — adding a non-Claude target becomes a new writer behind the materializer seam rather than a distribution rethink.

## References

- **G-0177** — the gap recording the friction and the agent-agnostic blocker this ADR decides.
- **ADR-0007** — *Planning-conversation skills: rituals-plugin placement, pure-skill default* — placement/authoring layering preserved; delivery-channel assumption revised here.
- **ADR-0006** — *Skills policy* — the complementary granularity axis; unaffected.
- **`docs/pocv3/plans/rituals-plugin-plan.md`** — the marketplace distribution design this supersedes.
- **CLAUDE.md** commitments #5 (marker-managed artifacts regenerated on `init`/`update`) and #6 (layered location-of-truth) — extended by this decision; § "What is *not* in scope" (no third-party skill registry / module system) — rules out runtime fetch; § "Cross-repo plugin testing" — the vendoring + drift-test pattern reused; § "Operator setup" — rewritten by the implementing epic.
- **Agent Skills standard** (agentskills.io) and **OpenAI Codex** (`.agents/skills/`) — evidence that SKILL.md is a cross-vendor open format, so portability is an output-location concern.
- **Go module proxy** — does not fetch git submodule contents, hence the vendored snapshot must be committed files.
