---
id: M-0096
title: Ship aiwfx-start-epic skill with worktree and branch preflight prompts
status: in_progress
parent: E-0028
depends_on:
    - M-0094
    - M-0095
tdd: required
acs:
    - id: AC-1
      title: Fixture exists with valid skill frontmatter and 10-step Workflow section
      status: met
      tdd_phase: done
    - id: AC-2
      title: Worktree-placement prompt is heading-scoped Q&A with three named options
      status: open
      tdd_phase: green
    - id: AC-3
      title: Sovereign-promotion step names the M-0095 rule and the override path
      status: open
      tdd_phase: red
    - id: AC-4
      title: Branch prompt is heading-scoped Q&A with G-0059 deferral note
      status: open
      tdd_phase: red
    - id: AC-5
      title: Drift-check test compares fixture to cache; skips cleanly when absent
      status: open
      tdd_phase: red
---

# M-0096 — Ship `aiwfx-start-epic` skill with worktree and branch preflight prompts

## Goal

Ship the `aiwfx-start-epic` ritual upstream in the `aiwf-extensions` rituals plugin. The skill orchestrates G-0063's preflight + sovereign promotion + optional delegation flow, with two new deliberate Q&A choices at start time: **worktree placement** and **branch shape**. Authored via the canonical fixture pattern (`internal/policies/testdata/aiwfx-start-epic/SKILL.md`) per CLAUDE.md; copied to the rituals repo at wrap; drift-checked against the local plugin cache.

## Context

M-0094 and M-0095 land the kernel chokepoints this skill relies on (`epic-active-no-drafted-milestones` finding; sovereign-act enforcement on `aiwf promote E-NN active`). With those in place, the skill's preflight has real signals to lean on instead of LLM-honor checks. The skill itself is the human-facing surface that closes G-0063's start-side concerns.

The worktree-placement prompt and the branch prompt are deliberately separate Q&A steps (Decision 4 of the planning conversation; recorded in the epic spec). The branch prompt is a placeholder pending G-0059's resolution — it asks rather than defaults, so the operator stays sovereign over the choice until a kernel-defaulted branch convention lands.

## Acceptance criteria

(ACs allocated at `aiwfx-start-milestone` time per the planner-skill convention.)

## Expected shape

- **Fixture-side authoring** — `SKILL.md` lives at `internal/policies/testdata/aiwfx-start-epic/SKILL.md` during the milestone; structural AC tests under `internal/policies/m0096_test.go` (or similar) assert content claims (presence of the worktree-placement section, the branch section, the delegation section, the sovereign-promotion step, the hand-off step). Per CLAUDE.md *Substring assertions are not structural assertions*, the assertions are heading-scoped, not flat greps.
- **Skill body** — the 10-step orchestration laid out in E-0028's scope. The worktree and branch prompts are explicit Q&A with numbered options (matching the project's existing Q&A convention).
- **Drift-check test** — asserts the fixture content matches the local marketplace cache (`~/.claude/plugins/cache/ai-workflow-rituals/.../SKILL.md`) when present; skips cleanly when absent. Matches the M-0090 precedent.
- **Wrap step** — at milestone wrap, copy the fixture content to `/Users/peterbru/Projects/ai-workflow-rituals/aiwf-extensions/skills/aiwfx-start-epic/SKILL.md` as a separate commit there; record that commit's SHA in this milestone's *Validation* section.
- **Follow-up gap at epic wrap** — file the wrap-side concerns (scope-end-before-done + ADR + `aiwfx-wrap-epic` update + human-only enforcement on `done`) as a gap referencing E-0028.

## Dependencies

- **M-0094** — `epic-active-no-drafted-milestones` finding. The skill's drafted-milestone preflight step consumes it.
- **M-0095** — sovereign-act enforcement on `aiwf promote E-NN active`. The skill's promotion step runs against the new refusal rule; the human-actor path is the default.

## References

- E-0028 epic spec — full skill orchestration laid out in *Scope → In scope → Rituals plugin skill*.
- G-0063 — gap framing, preflight checks, sub-decisions.
- M-0090 — precedent for cross-repo SKILL.md fixture + drift-check + wrap-time SHA recording.
- CLAUDE.md *Cross-repo plugin testing* — convention for SKILL.md authoring location.
- CLAUDE.md *AC promotion requires mechanical evidence* — structural assertions over fixture content.

### AC-1 — Fixture exists with valid skill frontmatter and 10-step Workflow section

### AC-2 — Worktree-placement prompt is heading-scoped Q&A with three named options

### AC-3 — Sovereign-promotion step names the M-0095 rule and the override path

### AC-4 — Branch prompt is heading-scoped Q&A with G-0059 deferral note

### AC-5 — Drift-check test compares fixture to cache; skips cleanly when absent

