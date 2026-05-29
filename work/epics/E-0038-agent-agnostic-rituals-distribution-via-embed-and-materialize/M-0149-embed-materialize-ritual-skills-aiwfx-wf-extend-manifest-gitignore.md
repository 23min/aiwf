---
id: M-0149
title: Embed + materialize ritual skills (aiwfx-/wf-); extend manifest + gitignore
status: done
parent: E-0038
depends_on:
    - M-0148
tdd: required
acs:
    - id: AC-1
      title: aiwf init writes embedded ritual skills to .claude/skills/{aiwfx,wf}-*
      status: met
      tdd_phase: done
    - id: AC-2
      title: Manifest and gitignore own the skill dirs; update refreshes, no clobber
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf check is clean against a repo materialized with ritual skills
      status: met
      tdd_phase: done
---
## Goal

Embed the vendored ritual skills (`aiwfx-*`, `wf-*`) into the engine binary via `go:embed` and extend the `init`/`update` materializer (plus the `.aiwf-owned` manifest and `.gitignore` patterns) so they are written into the consumer repo's `.claude/skills/` alongside the existing verb skills.

## Context

M1 vendored the snapshot. This milestone makes `aiwf init` / `aiwf update` actually deliver the ritual *skills* — the largest and most-used slice of the rituals — through the same marker-managed pipeline that already ships the 16 verb skills. After it lands, an operator gets the planning, lifecycle, and engineering skills with no `/plugin` step.

## Acceptance criteria

## Constraints

- Reuse the existing materializer / manifest / gitignore mechanism; do not fork a parallel path.
- Never clobber user-authored skills under `.claude/skills/` (the existing guarantee is preserved).
- Writes the Claude location directly — the target seam is M4, not this milestone.

## Design notes

- ADR-0014 §1 (build-time embed) and §3 (artifact coverage).
- CLAUDE.md commitment #5 (marker-managed artifacts regenerated on `init`/`update`) extended to ritual skills.

## Surfaces touched

- `internal/skills/` (embed directive + `Materialize`), `internal/initrepo/` (gitignore patterns), the `.aiwf-owned` manifest.

## Out of scope

- Agents and templates — M3.
- The agent-target abstraction — M4 (this milestone writes the Claude location directly).

## Dependencies

- M1 — the vendored snapshot to embed.

## References

- **ADR-0014** (§1, §3), **G-0177**, **E-0038**.

### AC-1 — aiwf init writes embedded ritual skills to .claude/skills/{aiwfx,wf}-*

### AC-2 — Manifest and gitignore own the skill dirs; update refreshes, no clobber

### AC-3 — aiwf check is clean against a repo materialized with ritual skills

## Work log

- **AC-1 — embed + materialize ritual skills.** Added `//go:embed embedded-rituals` + `ListRituals()` (walks `plugins/*/skills/*/SKILL.md`, flattening the plugin wrapper); `Materialize` now writes the union of verb + ritual skills into `.claude/skills/<name>/`. · tests: `TestListRituals`, `TestMaterialize_WritesRitualSkills`, `TestMaterialize_WritesVerbSkillsToo`
- **AC-2 — manifest + gitignore + no-clobber.** The `.aiwf-owned` manifest records the union; `GitignorePatterns()` gains `aiwfx-*/` and `wf-*/` (distinct prefixes — `aiwf-*` does not match `aiwfx-*`); re-materialize is idempotent; user-authored skill dirs are untouched. · tests: `TestMaterialize_ManifestOwnsRitualSkills`, `TestMaterialize_RitualsIdempotent`, `TestMaterialize_DoesNotClobberUserSkills`, `TestGitignorePatterns_CoverRituals`
- **AC-3 — check clean post-materialize.** `aiwf init` (which now materializes rituals) produces a tree that `check.Run` reports with zero error-severity findings. · tests: `TestInit_RitualSkillsMaterializedAndCheckClean`

The per-phase timeline (red→green→done→met) is the authoritative record in `aiwf history M-0149/AC-<N>`.

## Validation

- `go test ./...` — all packages pass, 0 failures (incl. the full `internal/policies` suite — the new embed does not trip `skill_coverage` or any other policy).
- `go vet ./internal/skills/ ./internal/initrepo/` — clean. `gofmt -l` — clean.
- **End-to-end binary smoke:** built `aiwf` (now embedding the rituals), ran `aiwf init` in a throwaway repo → materialized 29 skills (16 verb + 9 `aiwfx` + 4 `wf`); `aiwfx-plan-epic/SKILL.md` carries valid `name`/`description` frontmatter; `.gitignore` got the `aiwfx-*/` and `wf-*/` wildcards.
- Two existing tests (`TestGitignorePatterns`, `TestMaterialize_WritesManifest`) updated to the new union contract — intended behavior change, not papered over.

## Deferrals

- None at the milestone level. Agents + templates materialization is **M-0150**; the full in-this-repo install smoke (with de-dupe vs the marketplace plugin) is **M-0150/AC-4**.

## Reviewer notes

- **Flatten, don't nest.** Ritual skills materialize as `.claude/skills/<skill-name>/` (the `plugins/<plugin>/skills/` wrapper is dropped) — Claude discovers skills flat, matching the verb-skill layout.
- **Embed scope.** `//go:embed embedded-rituals` embeds the whole snapshot (skills + agents + templates), but `ListRituals` returns only files literally named `SKILL.md` under a `skills/` parent — agents/templates are inert until M-0150 materializes them. (Dotfiles like `.claude-plugin/plugin.json` and `templates/.gitkeep` are excluded by `go:embed`'s default dot-skip; irrelevant to skills.)
- **Defensive branches.** `ListRituals`'s `walkErr`/`readErr` guards and the `skills/`-parent check are defensive against embed states that can't arise from the vendored data (no `SKILL.md` lives outside a `skills/` dir); left in as cheap insurance rather than contrived into tests.
- **De-dupe is M-0152.** Materializing `aiwfx-*`/`wf-*` into `.claude/skills/` will collide with the marketplace plugin's same-named skills in a repo that has both — out of scope here (tests use temp dirs); the guard lands in M-0152.

