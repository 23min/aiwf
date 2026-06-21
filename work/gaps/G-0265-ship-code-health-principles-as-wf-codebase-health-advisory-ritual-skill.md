---
id: G-0265
title: Ship code-health principles as wf-codebase-health advisory ritual skill
status: open
---
Downstream aiwf consumers get the kernel's mechanical guarantees and the
ritual skills on `aiwf init` / `aiwf update`, but they do **not** get the
code-health field guide this repo scores itself against (the rubric behind
the periodic health scorecards under `docs/pocv3/`). The rubric is valuable
and reusable, yet it currently lives only as an in-repo doc with no
distribution vehicle.

## Decision

Ship the rubric as a new advisory **ritual skill**, `wf-codebase-health`,
in the `wf-rituals` plugin. It materializes into the consumer's
`.claude/skills/` on `init` / `update` exactly like the other `wf-*`
engineering skills — gitignored, marker-managed, versioned with the binary,
AI-discoverable. The assistant consults it when designing a module, planning
a refactor, reviewing a non-trivial diff, or running a scorecard.

This places the principles in aiwf's **advisory** half (rituals), not its
**mechanical** half (check rules / hooks). That is deliberate: the
principles are judgment forces, not pass/fail rules; mechanizing them would
be impossible for a language-agnostic tool and would prescribe how consumers
write code — which aiwf does not do.

## Vehicle — roads not taken

- **Inject into the per-turn guidance fragment** — rejected. That fragment
  is scoped to *operating aiwf*, explicitly "not about this project's own
  code"; code-health principles are the opposite, and per-turn context
  should stay lean.
- **Opt-in materialization of a tracked doc** into the consumer's `docs/`
  — deferred (YAGNI). aiwf does not write tracked files without explicit
  per-invocation consent; if a consumer wants the rubric committed for human
  readers, that is a future consent-gated knob, not part of this change.
- **A `check` rule or git hook over consumer code** — rejected. Prescribes,
  cannot be language-agnostic, and removes the freedom this design preserves.

## Scope of the closing patch

- `wf-codebase-health/SKILL.md` (embedded; auto-picked-up by the
  `embedded-rituals` tree embed).
- One-line pointers from `wf-review-code` and the `reviewer` agent
  (per-diff gate to whole-codebase rubric).
- A structural test asserting the skill materializes and carries the
  principle sections.
- A `CLAUDE.md` line in the engineering-skills list.
- A companion ADR recording the vehicle decision above.
