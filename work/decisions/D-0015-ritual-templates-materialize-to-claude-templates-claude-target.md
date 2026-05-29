---
id: D-0015
title: Ritual templates materialize to .claude/templates/ (Claude target)
status: proposed
---
## Decision

Ritual **templates** (`adr.md`, `decision.md`, `epic-spec.md`,
`milestone-spec.md`) materialize to **`.claude/templates/<name>.md`** for the
Claude target — a sibling of `.claude/skills/` and `.claude/agents/`. Each of
the three artifact roots carries its own `.aiwf-owned` ownership manifest,
mirroring the skills wipe-and-rewrite mechanism exactly.

## Context

ADR-0014 §3 enumerates three materializable ritual artifact kinds and pins two
destinations concretely — skills → `.claude/skills/{aiwfx,wf}-*/SKILL.md`,
agents → `.claude/agents/*.md` — but leaves templates as "→ their referenced
locations," deliberately underspecified.

Today (marketplace path) templates are not in the consumer repo at all: they
live machine-local in the plugin cache at
`~/.claude/plugins/cache/ai-workflow-rituals/aiwf-extensions/<sha>/templates/`,
co-located under the same per-version plugin root as the skills that reference
them. The ritual skills point at "this plugin's `templates/X.md`" — a
prose hint the agent reads, resolving as a sibling dir under the shared plugin
root. Embed-and-materialize flattens skills to `.claude/skills/<name>/`, which
breaks that co-location, so templates need an explicit consumer-repo home.

## Options weighed

1. **`.claude/templates/`** (chosen) — sibling of skills/agents. Symmetric:
   three artifact kinds → three Claude dirs, matching ADR-0014 §3's framing.
   Cleanest conceptual home; own manifest, no migration of M-0149's manifest.
2. `.claude/skills/.aiwf-templates/` — co-located under the skills dir, closer
   to today's physical layout; reuses the skills gitignore. Rejected: mixes
   templates into the skills tree and stretches the single manifest.

## Consequences

- The embedded skill bodies are **not** rewritten to point at the new path —
  they are a drift-checked verbatim snapshot of upstream (M-0148's
  `TestRituals_VendoredMatchesUpstream`), so editing them would fail the drift
  guard. The agent reads "templates/X.md" as a name hint and finds the
  materialized file by basename in `.claude/templates/`.
- Output location is "just a parameter" per ADR-0014 §4; M-0151's agent-target
  seam will parameterize it per target (Codex etc.), so this is the Claude
  target's value, not a universal constant.
- Low reversal cost: everything here is gitignored and regenerated on
  `init`/`update`.

## References

- **ADR-0014** §3 (artifact coverage), §4 (agent-target seam) — this resolves §3's underspecified template location.
- **M-0150** — the implementing milestone.
- **M-0148** — the vendored-snapshot drift guard that forbids editing embedded skill bodies.
