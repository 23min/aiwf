---
id: G-0580
title: The skill-edit backstop reaches neither agent cards nor verb skills
status: open
priority: low
---
## What's missing

The skill-edit provenance backstop
(`internal/policies/skill_edit_provenance_backstop.go`) fails the profile-driven
gate when a commit changes a `SKILL.md` under the embedded rituals and carries no
`aiwf-entity:` trailer naming an entity in the tree (D-0071). Its scope is exactly
that: the embedded-rituals subtree, and a filename of `SKILL.md`.

Two classes of shipped surface fall outside it. Role-agent cards, which live beside
the ritual skills but are not named `SKILL.md`. And the verb skills under
`internal/skills/embedded/`, which are `SKILL.md` files but are not under the
embedded-rituals subtree. Both materialize into a consumer repo on `aiwf update`.

Measured at `3c5bd7546` (go1.25.12, Linux) in a scratch clone: commit a one-line
edit, then run
`AIWF_COVERAGE_BASE=<the commit before it> go test -count=1 -v -run '^TestPolicy_SkillEditProvenanceBackstop$' ./internal/policies/`.

- `internal/skills/embedded-rituals/plugins/wf-rituals/skills/wf-patch/SKILL.md`,
  no trailer: `--- FAIL`, "edits this shipped ritual SKILL.md but carries no
  aiwf-entity: trailer".
- The same edit carrying `aiwf-entity: G-0580`: `--- PASS`.
- `internal/skills/embedded-rituals/plugins/aiwf-extensions/agents/builder.md` and
  `internal/skills/embedded/aiwf-add/SKILL.md` in one commit, no trailer:
  `--- PASS`.

## Why it matters

The backstop exists because an edit to a shipped surface reaches consumers directly,
and an edit no entity owns is how consumer-facing prose drifts unobserved. That
reasoning applies identically to an agent card and to a verb skill; only the path
glob distinguishes them.

It is not theoretical. On 2026-08-11 both uncovered classes carried drifted
body-scaffold instructions, and the drift had survived a prior repair pass precisely
because that pass was scoped to skills — the same scoping the backstop has.

Widening the glob is the obvious option and is not obviously right: it would impose
the mandate on more files, and a mandate costs per subject forever. Whether the
answer is a wider glob, a different backstop shape, or nothing at all is the decision
to make.
