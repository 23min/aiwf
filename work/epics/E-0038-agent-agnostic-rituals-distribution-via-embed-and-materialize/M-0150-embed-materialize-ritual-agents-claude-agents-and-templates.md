---
id: M-0150
title: Embed + materialize ritual agents (.claude/agents/) and templates
status: done
parent: E-0038
depends_on:
    - M-0149
tdd: required
acs:
    - id: AC-1
      title: aiwf init materializes ritual agents to .claude/agents/ and templates
      status: met
      tdd_phase: done
    - id: AC-2
      title: Manifest owns agents and templates; update refreshes without clobber
      status: met
      tdd_phase: done
    - id: AC-3
      title: Test asserts no new hook surface beyond aiwf's existing git hooks
      status: met
      tdd_phase: done
    - id: AC-4
      title: make install + aiwf update materializes rituals into .claude/, human-verified
      status: deferred
      tdd_phase: red
---
## Goal

Extend the embed+materialize pipeline to the rituals' agents (→ `.claude/agents/`) and templates (→ their referenced locations), with manifest ownership and gitignore coverage, treating agents exactly like skills.

## Context

M2 delivers ritual skills. The rituals also ship four agents (`planner` / `builder` / `reviewer` / `deployer`) and a set of templates. This milestone completes artifact coverage so `aiwf init` delivers the full ritual set. Hooks are explicitly *not* part of the rituals (ADR-0014 §3), so no hook surface is added.

## Acceptance criteria

## Constraints

- Agents are materialized like skills — same manifest ownership and gitignore discipline; user-authored agents are never clobbered.
- No new hook installation (ADR-0014 §3). The only managed hooks remain aiwf's existing git hooks.

## Design notes

- ADR-0014 §3 (artifact coverage; agents-as-skills; hooks-not-rituals).
- **D-0015** — ritual templates materialize to `.claude/templates/<name>.md` for the Claude target (resolves ADR-0014 §3's deliberately-open "→ their referenced locations"). Sibling of `.claude/skills/` and `.claude/agents/`; three artifact kinds → three Claude dirs.

## Surfaces touched

- `internal/skills/` (embed + materialize for agents/templates), `internal/initrepo/`, the manifest.

## Out of scope

- Per-target agent handling for non-Claude agents — M4.
- The marketplace sunset — M5.

## Dependencies

- M2 — the materializer extended for skills, which this milestone extends further.

## References

- **ADR-0014** (§3), **E-0038**.

### AC-1 — aiwf init materializes ritual agents to .claude/agents/ and templates

### AC-2 — Manifest owns agents and templates; update refreshes without clobber

### AC-3 — Test asserts no new hook surface beyond aiwf's existing git hooks

### AC-4 — make install + aiwf update materializes rituals into .claude/, human-verified

**Deferred → M-0152/AC-4.** The live-repo install smoke can only be verified cleanly after M-0152's de-dupe guard tells the operator to disable the still-enabled marketplace plugin; verifying it here would create the duplicate-skill collision M-0152 exists to detect, and leaving it open on M-0150 would deadlock wrap since M-0152 `depends_on` M-0150. Relocated as `M-0152/AC-4`.

## Work log

### AC-1 — materialize ritual agents + templates
Added `AgentsDir`/`TemplatesDir` consts, `ListRitualAgents`/`ListRitualTemplates` (shared `listRitualFiles` walker), and extended `Materialize` to write agents → `.claude/agents/` and templates → `.claude/templates/` via a new `materializeFlatFiles` helper. · tests: `TestListRitualAgents`, `TestListRitualTemplates`, `TestMaterialize_WritesRitualAgents`, `TestMaterialize_WritesRitualTemplates`, `TestInit_RitualAgentsAndTemplatesMaterialized` (init seam)

### AC-2 — manifest ownership + gitignore + no-clobber
Each flat root carries its own `.aiwf-owned` manifest (same wipe-and-rewrite contract as skills). `GitignorePatterns()` now returns `([]string, error)` and enumerates the agent/template files + per-root manifests (enumeration, not a wildcard, because these basenames have no namespacing prefix). · tests: `TestMaterialize_ManifestOwnsAgentsAndTemplates`, `TestMaterialize_AgentsTemplatesIdempotent`, `TestMaterialize_DoesNotClobberUserAgents`, `TestMaterialize_FlatFilesWipeRemoved`, `TestGitignorePatterns_CoverAgentsAndTemplates`, `TestInit_GitignoreCoversAgentsAndTemplates` (ensureGitignore seam)

### AC-3 — no new hook surface
Structural guard asserting the embed ships no hook artifact and materialization writes none under `.claude/` (ADR-0014 §3). Verified the guard fires via plant-and-revert of a synthetic `hooks.json`. · tests: `TestRituals_NoHookSurface`

The per-phase red→green→done→met timeline is authoritative in `aiwf history M-0150/AC-<N>`.

## Decisions made during implementation

- **D-0015** — ritual templates materialize to `.claude/templates/` for the Claude target. Resolves ADR-0014 §3's open template location; the embedded skill bodies are *not* rewritten to point at the path (they are a drift-checked verbatim upstream snapshot — M-0148's guard forbids it), so the agent resolves "templates/X.md" by basename.

## Validation

- `go build ./...` clean; `go test ./...` — all packages pass, 0 failures (incl. `internal/policies`).
- `go vet ./internal/skills/ ./internal/initrepo/` clean; `gofmt -l` clean.
- **Branch coverage:** every logic branch in the new code is exercised (wipe-vs-continue, no-clobber, write, empty-manifest first run, gitignore enumeration). The only uncovered lines are error-return arms of operations that cannot fail against the in-memory embed or a fresh `TempDir` (walk/read/mkdir/remove/write) — the same defensive-wrap pattern `ListRituals` (80%) and `Materialize` (71.4%) already carry; not contrived into fault-injection tests, consistent with M-0149.
- **Binary smoke** (real `aiwf init` in a throwaway repo): materialized 4 agents (`builder/deployer/planner/reviewer.md`) → `.claude/agents/` and 4 templates (`adr/decision/epic-spec/milestone-spec.md`) → `.claude/templates/`, both `.aiwf-owned` manifests correct, all 10 gitignore lines present, `git status` confirms the artifacts are ignored, zero hooks materialized.
- Three existing tests (`TestGitignorePatterns`, `TestGitignorePatterns_BinaryEntryListed`, `TestGitignorePatterns_CoverRituals`) updated to the new `([]string, error)` signature — intended contract change, not papered over.

## Deferrals

- **AC-4** (live-repo install smoke) → relocated to **M-0152/AC-4**, sequenced after the de-dupe guard. See the AC-4 note above.

## Reviewer notes

- **Flat vs. dir layout.** Skills materialize as `<name>/SKILL.md` (dir-per-skill); agents and templates are flat single files (`<name>.md`), so they get a parallel `materializeFlatFiles` path rather than reusing the dir-based skills loop. Two shapes, one ownership/manifest contract.
- **No namespacing prefix.** Agent/template basenames (`builder.md`, `adr.md`) have no `aiwf`-style prefix, so the gitignore enumerates exact paths instead of a wildcard — a directory wildcard would mask user-authored files. `ensureGitignore` reconciles on every `init`/`update`, so a future ritual addition's line is appended by the same `update` that materializes it.
- **`GitignorePatterns` signature change.** Now returns an error because it derives the agent/template names from the embed (so an upstream rename can't silently desync the gitignore). Hardcoding the names would have kept the old signature but reintroduced the drift risk.
- **Skill prose unchanged.** The embedded skills still say "this plugin's `templates/X.md`"; we deliberately do not rewrite them (drift guard), and the agent resolves by basename in `.claude/templates/`.
- **Defensive coverage.** Error-return branches on infallible-against-embed operations are left uncovered by design — see Validation.

