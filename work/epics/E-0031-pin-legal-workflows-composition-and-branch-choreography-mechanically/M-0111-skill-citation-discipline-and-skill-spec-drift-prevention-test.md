---
id: M-0111
title: Skill-citation discipline and skill-spec drift-prevention test
status: draft
parent: E-0031
depends_on:
    - M-0108
tdd: required
---
## Goal

Add citation links from each `aiwfx-*` and `wf-rituals:*` skill body to its corresponding workflow in `legal-workflows.md`. Add an `internal/policies/` test that pins the skill ↔ spec correspondence — every spec workflow has a citing skill; every skill workflow is in the spec.

## Context

M-0108 ships the spec; this milestone makes skills citizens of the spec rather than duplicators of it. Drift-prevention is the chokepoint that makes "spec is source of truth" (E-0031 constraint) survive refactors. Can run parallel with M-0109/M-0110.

## Approach

Each skill body gets a "**Spec:** see `docs/pocv3/design/legal-workflows.md#<workflow-anchor>`" line near its "## When to use" section. The policy test (`internal/policies/skill_spec_drift.go`) walks `.claude/skills/aiwfx-*/SKILL.md` and `wf-rituals:*/SKILL.md`, extracts the spec link, asserts each named workflow exists in the spec, and asserts every spec workflow has at least one citing skill. Fails CI if either side drifts.

## Acceptance criteria

<!-- ACs are added at aiwfx-start-milestone via `aiwf add ac M-0111 --title "..."`. -->

## Surfaces touched

- `.claude/skills/aiwf-*/SKILL.md` (each gets a citation line)
- `internal/policies/skill_spec_drift.go` (new)
- `internal/policies/skill_spec_drift_test.go` (new)

## Out of scope

- Test harness work (M-0109/M-0110)
- Fuzz harness (M-0112)

## Dependencies

- M-0108 (spec must exist for skills to cite)
