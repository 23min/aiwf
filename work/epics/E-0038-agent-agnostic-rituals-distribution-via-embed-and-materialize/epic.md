---
id: E-0038
title: Agent-agnostic rituals distribution via embed-and-materialize
status: proposed
---
## Goal

Make `aiwf` itself the distribution mechanism for the rituals — vendor a pinned snapshot into the aiwf repo, embed it, and materialize it on `aiwf init` / `aiwf update` — so a consumer gets the planning skills, lifecycle rituals, agents, and templates with **one command and no `/plugin` step**, and so adding a non-Claude agent target later is a new writer rather than a distribution rethink. Retire the Claude marketplace channel once the embedded path is stable. Implements ADR-0014; addresses G-0177.

## Context

aiwf ships two layers of advisory artifact. Verb skills (`aiwf-*`) are embedded in the binary and materialized by `init`/`update` — friction-free. Rituals (`aiwfx-*`/`wf-*` skills, the `planner`/`builder`/`reviewer`/`deployer` agents, templates) are distributed as an external Claude marketplace plugin (`23min/ai-workflow-rituals`), installed manually at project scope with no auto-update (ADR-0007, `docs/pocv3/plans/rituals-plugin-plan.md`).

That marketplace channel is both high-friction (manual install, no upgrade, Claude-plugin-index path fragility per anthropics/claude-code#31388) and a hard blocker on the agent-agnostic future: it is Claude-only by construction. The skill *format* is already portable — SKILL.md is a cross-vendor open standard (agentskills.io; Codex reads the identical frontmatter from `.agents/skills/`) — so the obstacle is the delivery channel, not the file. ADR-0014 decides the fix: build-time embed of a vendored snapshot, materialized like the verb skills, behind an agent-target seam, with a phased marketplace sunset.

This epic carries that work. Per CLAUDE.md § "Authoring an ADR", the decision lives in ADR-0014; this epic sequences the action.

## Scope

### In scope

- A **vendor-sync mechanism** that pulls a pinned upstream rituals SHA into a path in the aiwf repo as **real committed files** (not a submodule — the Go module proxy does not fetch submodule contents), plus a drift-check test mirroring the existing cross-repo SKILL.md fixture discipline.
- **Embedding** the vendored ritual skills, agents, and templates via `go:embed`.
- **Extending the materializer** (and the `.aiwf-owned` manifest + `.gitignore` patterns) so `init`/`update` write ritual skills to `.claude/skills/{aiwfx,wf}-*`, agents to `.claude/agents/`, and templates to their referenced locations — preserving the never-clobber-user-skills guarantee.
- An **agent-target abstraction** in the materializer: a target parameter with the Claude writer implemented and a seam for additional targets. (Implementing a second target is optional within this epic — see Open questions.)
- **Marketplace sunset**: flip `aiwf doctor` from recommending the plugin to verifying materialized artifacts; drop `doctor.recommended_plugins` from the default `aiwf.yaml`; a **de-dupe guard** that detects an enabled marketplace plugin (via `.claude/settings.json` `enabledPlugins`) and instructs the operator to disable it rather than silently editing settings.
- Rewriting CLAUDE.md § "Operator setup" to the one-command flow; updating/archiving `rituals-plugin-plan.md`.

### Out of scope

- **Runtime network fetch** of the rituals — ADR-0014 chooses build-time embed; a remote skill registry is out of scope per CLAUDE.md.
- **An MCP server exposing the verbs** — a complementary cross-agent track for verb *execution*, not skill distribution. Separate epic if pursued.
- **The project-prose portability layer** (`CLAUDE.md` vs `AGENTS.md`) — a different surface; separate concern.
- **New ritual content** — this is a distribution change. Authoring stays upstream.
- **Full implementation of every non-Claude target** — the seam is in scope; building out Codex/Cursor/Copilot writers beyond a proof is deferred.

## Constraints

- **Vendored snapshot = committed files, not a submodule.** Load-bearing: a submodule embeds empty under `go install`. (ADR-0014 §2.)
- **Decision is decision; this epic sequences action.** No gate language migrates into ADR-0014; sequencing lives here.
- **Preserve `wf-*` aiwf-agnostic reuse.** The upstream rituals repo stays the authoring home; aiwf only vendors a snapshot.
- **AC promotion requires mechanical evidence** (CLAUDE.md) — tests under `internal/policies/`, kernel finding-rules, or fixture-validation, even for `tdd: none` milestones. Materialization, manifest ownership, gitignore coverage, and the de-dupe guard are all mechanically assertable.
- **Never clobber user-authored skills/agents.** The existing manifest discipline extends to the new dirs.
- **No half-finished implementations.** If the embed+materialize milestone lands, `aiwf init` on a fresh repo produces a working ritual set and `aiwf check` is clean.

## Success criteria

<!-- Observable outcomes at epic close, not tests. -->

- [ ] `aiwf init` on a fresh consumer repo installs verb skills **and** rituals (skills + agents + templates) with no `/plugin` step.
- [ ] `aiwf update` after a binary upgrade refreshes the rituals to the version vendored in that binary.
- [ ] The vendored rituals snapshot is present as committed files in the aiwf repo with its pinned upstream SHA recorded; a drift-check test fails if the snapshot diverges from the pinned upstream.
- [ ] No new hook surface is introduced — only aiwf's existing git hooks are managed (asserted mechanically).
- [ ] The materializer exposes an agent-target seam; the Claude target is implemented; a non-Claude target is either spiked as a proof or explicitly deferred via a filed gap.
- [ ] `aiwf doctor` no longer recommends the marketplace plugin and instead verifies materialized artifacts; the de-dupe guard fires with operator guidance when both the plugin and materialized artifacts are present.
- [ ] CLAUDE.md § "Operator setup" describes the one-command flow; `rituals-plugin-plan.md` is updated or archived to reflect the retired channel.
- [ ] ADR-0014 is referenced by the implementing milestones; G-0177 is promoted to `addressed` and archived under this epic's wrap.

## Open questions

| Question | Blocking? | Resolution path |
|---|---|---|
| Vendor-sync implementation: `git subtree` pull vs scripted copy vs `go:generate` fetch-at-build? | no | Decided at the vendor-sync milestone; default lean: scripted copy + committed snapshot + drift test, matching the cross-repo fixture pattern. |
| Where the pinned upstream SHA is recorded (a `rituals.lock` file, a Makefile var, the sync commit trailer)? | no | Decided at the vendor-sync milestone. |
| Does this epic ship a second (non-Claude) target as proof, or only the seam? | no | Decided after the materializer milestone, based on remaining budget; if deferred, file a gap. |
| For non-Claude targets, do agents materialize or no-op? | no | Per-target writer decision; Claude materializes agents, targets without a subagent concept no-op. |
| Hard-remove the marketplace, or keep it as an alternate install path during a transition window? | no | Decided at the sunset milestone; default lean: deprecate-then-remove across two releases. |

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Overlap window: both marketplace plugin and materialized artifacts present → duplicate skill `name:` collisions. | medium | The de-dupe guard detects an enabled plugin and instructs the operator to disable it before/after migrating. |
| Vendored snapshot drifts from upstream silently. | medium | Drift-check test pinned to the upstream SHA; fails CI on divergence (cross-repo fixture discipline). |
| Binary size growth from embedding. | low | Rituals are a few hundred KB of markdown; the 16 verb skills already embed without notice. |
| Upstream agent/skill format churn (Codex/Cursor) breaks a target writer. | low–medium | The agent-target seam isolates per-target format logic; only the affected writer changes. |
| Loss of independent ritual versioning surprises a consumer mid-cycle. | low | Documented in ADR-0014 consequences; pinned SHA is the provenance record; `aiwf upgrade` is the single refresh path. |

## Proposed milestones

> Not yet allocated — decomposition is the next step (`aiwfx-plan-milestones` against this epic). Listed here as the planned sequence, not as existing entities.

| # | Title | Depends on |
|---|-------|-----------|
| 1 | Vendor-sync mechanism: pull pinned rituals snapshot into the aiwf repo + drift test | — |
| 2 | Embed + materialize ritual skills (`aiwfx-*`/`wf-*`); extend manifest + gitignore; coverage tests | 1 |
| 3 | Embed + materialize agents (`.claude/agents/`) + templates | 2 |
| 4 | Agent-target abstraction in the materializer (Claude target implemented; seam for others) | 2 |
| 5 | Marketplace sunset: `doctor` flip, de-dupe guard, docs/CLAUDE.md rewrite, drop `recommended_plugins` default | 3, 4 |
| 6 (optional) | Non-Claude target proof (e.g. Codex `.agents/skills/`) or a deferral gap | 4 |

## Supersedes / addresses

- **G-0177** — the friction + agent-agnostic-blocker gap. Promoted to `addressed` and archived at this epic's wrap.
- **`docs/pocv3/plans/rituals-plugin-plan.md`** — its marketplace distribution design is superseded; updated or archived by milestone 5.

## References

- **ADR-0014** — *Embed-and-materialize rituals distribution; retire Claude marketplace* — the decision this epic implements.
- **G-0177** — the motivating gap.
- **ADR-0007** — placement/authoring layering preserved; only its delivery-channel assumption is revised by ADR-0014.
- **CLAUDE.md** commitments #5 and #6, § "Operator setup", § "Cross-repo plugin testing", § "Authoring an ADR", § "AC promotion requires mechanical evidence".
- **G-0175** — sibling rituals-distribution concern; may be revisited as part of the materialized-ritual trailer story.
