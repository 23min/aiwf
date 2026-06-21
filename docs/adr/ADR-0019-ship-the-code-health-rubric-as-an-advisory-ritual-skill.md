---
id: ADR-0019
title: Ship the code-health rubric as an advisory ritual skill
status: proposed
---
## Context

aiwf scores its own codebase against a stack-agnostic field guide,
"Principles for a healthy codebase" — the rubric behind the periodic health
scorecards under `docs/pocv3/`. The rubric is reusable by any project, but it
has no distribution vehicle: a downstream consumer who runs `aiwf init` /
`aiwf update` receives the kernel's mechanical guarantees and the ritual
skills, but not the rubric.

The question is *what vehicle* carries the rubric to consumers, under two
standing constraints:

- aiwf writes freely only into gitignored, marker-managed locations
  (`.claude/` skills/agents/templates, `.git/hooks/`) plus a single
  marker-wrapped import line in the consumer's root `CLAUDE.md`. It does
  **not** write tracked files (e.g. into `docs/`) without explicit
  per-invocation consent (ADR-0015 / ADR-0018).
- aiwf does not prescribe how consumers write code. The rubric is a set of
  judgment *forces*, not pass/fail rules.

aiwf already ships two kinds of payload: **guarantees** (mechanical
chokepoints — `aiwf check`, the pre-push hook, the `internal/policies/`
tests) and **rituals** (advisory skills the assistant consults —
`wf-review-code`, `wf-tdd-cycle`, `wf-rethink`). The code-health rubric is
the same genus as the existing `wf-*` engineering skills.

## Decision

Ship the rubric as a new advisory **ritual skill**, `wf-codebase-health`, in
the `wf-rituals` plugin under `internal/skills/embedded-rituals/`. It is
embedded via `go:embed` and materialized into the consumer's
`.claude/skills/` by `aiwf init` / `aiwf update`, on the same gitignored,
marker-managed, version-pinned pipeline as the other rituals (ADR-0014).
`wf-review-code` and the `reviewer` agent carry a one-line pointer to it.

The skill is **advisory, not mechanical**. It is not a `check` rule, not a
hook, and adds no kernel surface. This is the central decision:

- The principles are judgment forces; mechanizing them is impossible for a
  language-agnostic tool and would prescribe how consumers write code.
- A skill is consulted when relevant and can be ignored, edited, or deleted
  by the consumer — it preserves freedom by construction.
- Skill-as-advisory is consistent with "framework correctness must not depend
  on LLM behavior": no guarantee is claimed, so no chokepoint is owed.

### Roads not taken

- **Inject the rubric into the per-turn guidance fragment**
  (`aiwf-guidance.md`). Rejected: that fragment is scoped to *operating
  aiwf*, explicitly "not about this project's own code," and per-turn context
  must stay lean.
- **Opt-in materialization of a tracked rubric doc** into the consumer's
  `docs/`. Deferred (YAGNI). aiwf does not write tracked files without
  explicit consent; a consent-gated `aiwf.yaml` knob is a future option if a
  consumer wants the rubric committed for human readers. The advisory skill
  covers the assistant-facing need today.
- **A `check` rule or git hook over consumer code.** Rejected — prescribes,
  cannot be language-agnostic, and removes the freedom this design preserves.

## Consequences

- A new `wf-*` skill ships in every materialized `.claude/`: one more file in
  the embed set, auto-discovered by the `embedded-rituals` tree embed, and
  asserted by a structural test.
- The rubric is stack-agnostic and ships verbatim; its closing line invites
  per-codebase editing, and the consumer's own conventions win where they
  conflict.
- aiwf now distributes a *code-style opinion*, which narrows the
  "opinion-free kernel" framing — but only in the advisory layer, which
  already carries opinions (`wf-tdd-cycle`, `wf-rethink`). The mechanical
  layer stays opinion-free.
- The in-repo authoring doc
  (`docs/pocv3/design/healthy-codebase-principles.md`) and the embedded
  `SKILL.md` are two roles of the same content — authoring reference vs.
  distributable form — mirroring the source-vs-render pattern aiwf already
  uses for `ROADMAP.md` and the embedded rituals.
