---
id: M-0190
title: Default the start rituals to in-repo worktree placement
status: draft
parent: E-0046
depends_on:
    - M-0189
tdd: none
---

# M-0190 — Default the start rituals to in-repo worktree placement

## Goal

Make `aiwfx-start-epic` and `aiwfx-start-milestone` default to in-repo worktree placement
(reading the `worktree.dir` knob), with the per-invocation override retained and the
devcontainer-sandbox rationale recorded inline.

## Acceptance criteria

_Scaffolded via `aiwf add ac` at start-milestone (doc-shaped — structural assertions on
the embedded SKILL.md sections). Intended shape: (1) the start rituals' worktree-placement
step defaults to in-repo under the configured `worktree.dir`; (2) the per-invocation
override (main-checkout / sibling) stays selectable; (3) the sandbox rationale is present
and references the epic's ADR._

## Context

Builds on the `worktree.dir` knob (M-0189) and the loader guard (M-0188). The start
rituals currently offer three placements as a free choice with no default; this milestone
flips the recommended default to in-repo and records why. Authoring is in the embedded
ritual snapshot (`internal/skills/embedded-rituals/…`) per CLAUDE.md "Ritual content
authoring"; AC tests assert against the embedded bytes.

## Constraints

- The knob sets the *default*, not a lock — the per-invocation override stays.
- Doc-shaped ACs use structural assertions scoped to the named SKILL.md section, not flat
  substring greps (CLAUDE.md "Substring assertions are not structural assertions").

## Design notes

- References the epic's in-repo-worktree-default ADR (allocated in this epic); the real
  ADR id is wired in when the ADR lands.

## Out of scope

- The config knob (M-0189); the loader guard (M-0188).

## Dependencies

- M-0189 — the `worktree.dir` knob the rituals read.

## References

- E-0046 epic spec; CLAUDE.md "Ritual content authoring".
