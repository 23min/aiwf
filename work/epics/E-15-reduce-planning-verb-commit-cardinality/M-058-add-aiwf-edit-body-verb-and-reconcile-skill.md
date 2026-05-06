---
id: M-058
title: Add aiwf edit-body verb and reconcile skill
status: in_progress
parent: E-15
acs:
    - id: AC-1
      title: aiwf edit-body verb exists; accepts --body-file or stdin
      status: met
    - id: AC-2
      title: edit-body produces single trailered commit (aiwf-verb edit-body)
      status: met
    - id: AC-3
      title: aiwf-add skill text removes plain-git body-edit carve-out
      status: met
    - id: AC-4
      title: G-051 and G-052 are promotable to addressed after this milestone
      status: met
---

## Goal

Cover the post-creation body-edit case so the `aiwf-add` skill can drop its plain-git body-edit carve-out. After this milestone, every entity-file change goes through a verb route and `provenance-untrailered-entity-commit` only fires on accidental hand-edits. This is the milestone that lets G-052 close as `addressed`.

## Approach

A new verb `aiwf edit-body <id> --body-file <path>` (and `--body -` for stdin) reads the body content, replaces everything below the frontmatter line in the target entity file, validates the projected tree, and commits with the standard trailer block (`aiwf-verb: edit-body`, `aiwf-entity: <id>`, `aiwf-actor: <actor>`). Frontmatter is left untouched — that stays the domain of structured-state verbs like `promote` / `rename` / `cancel`.

Skill text update is a one-line change to `.claude/skills/aiwf-add/SKILL.md`: remove the "body-prose edits to an existing entity file ... [are allowed]" sentence, replace with a pointer to `aiwf edit-body`. Rides along with the verb implementation in the same milestone.

## Acceptance criteria

### AC-1 — aiwf edit-body verb exists; accepts --body-file or stdin

### AC-2 — edit-body produces single trailered commit (aiwf-verb edit-body)

### AC-3 — aiwf-add skill text removes plain-git body-edit carve-out

### AC-4 — G-051 and G-052 are promotable to addressed after this milestone

