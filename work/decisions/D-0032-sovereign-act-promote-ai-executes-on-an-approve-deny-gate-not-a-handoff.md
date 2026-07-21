---
id: D-0032
title: 'Sovereign-act promote: AI executes on an approve/deny gate, not a handoff'
status: accepted
---

# D-0032 — Sovereign-act promote: AI executes on an approve/deny gate, not a handoff

> **Date:** 2026-07-07 · **Decided by:** human/peter

## Question

`aiwf promote E-NN active` is a sovereign-act-shape transition (M-0095): the kernel refuses it from a non-`human/` actor unless `--force --reason "..."` is used. Since actor resolution defaults to `human/<git-email-localpart>` whenever no `--actor` flag is passed, the kernel already accepts this verb from an AI-run bare command indistinguishably from a human-run one — there is no mechanical way for the kernel to tell the two apart. The only thing currently stopping an AI assistant from running it is a documented ritual-policy line in `aiwfx-start-epic`'s `SKILL.md` ("the operator is human; an AI assistant orchestrating the conversation does not invoke the verb itself"), echoed in `CLAUDE.md` and `provenance-model.md`.

Should the assistant continue to hand the bare command off to the human to type themselves, or should it present the exact command as an explicit approve/deny gate and execute it directly once the human approves?

## Decision

Sovereign-act-shaped verbs (starting with `aiwf promote E-NN active`) present the exact command to the human as an explicit approve/deny gate. On explicit approval, the AI assistant executes the command directly (no `--actor` override — the bare command already resolves to the human's own git identity). The human's approval *is* the sovereign act; the AI is the executing tool under direct, per-invocation human authorization — the same shape as every other mutating gate this repo's rituals already use (`wf-patch`'s commit/merge/push gates, `aiwfx-release`'s five gates).

This reverses the current `aiwfx-start-epic` policy line and its echoes in `CLAUDE.md`'s provenance bullet and `provenance-model.md`, which state that an AI assistant orchestrating the conversation does not invoke the verb itself.

## Reasoning

The alternative — keep requiring the human to type the command themselves — was rejected for three reasons:

- **It isn't actually a stronger guarantee today.** The kernel enforcement (`internal/verb/promote_sovereign_act.go`) only distinguishes actors by the literal string prefix, and defaults to `human/...` whenever no `--actor` override is passed. The current "AI does not invoke it" behavior holds only because the AI reads and complies with a ritual-skill sentence — exactly the class of soft, LLM-behavior-dependent guarantee `CLAUDE.md` itself warns against elsewhere ("a guarantee that depends on the LLM remembering to invoke a skill is not a guarantee"). An explicit approve/deny gate is a *more* visible, more reliably-enforced checkpoint than silent doc compliance, not a weaker one.
- **It's inconsistent with the rest of the framework.** Every other consequential, hard-to-reverse action in this repo's rituals — commits, merges to mainline, tags, pushes — already uses "AI proposes the exact command, human clicks approve, AI executes." Singling out epic-activation for a stricter "must be human-typed" bar, while allowing e.g. a tag push (arguably at least as consequential) to go through a click-to-approve gate, has no principled basis once the kernel-level actor-resolution behavior is understood.
- **The friction cost was real and reported.** An operator asking the assistant to activate an epic, being told to type the command themselves, is the exact "occasional friction... every kernel-refusal-that-needs-overriding becomes a synchronous prompt" scenario G-0023 already names as plausible and worth watching for.

Alternative considered and rejected: extend `aiwf authorize --allow-force` (G-0023) to let a delegated agent invoke `--force` within an authorized scope. Rejected as the wrong tool for this case — it targets a different scenario (a long-running autonomous authorized scope), not the ordinary conversational "human directs, AI executes with per-step approval" case this decision covers, where no authorize scope exists or is needed and the actor legitimately resolves to `human/...` already.

## Consequences

- `aiwfx-start-epic/SKILL.md`'s promotion step (and its "Anti-patterns" and "Constraints" sections) needs rewriting: replace "hand off to the human to type it" with "present the exact command as an approve/deny gate; execute directly on approval." Per this repo's ritual-authoring convention, this edit needs a companion structural test under `internal/policies/` (the `skill-edit-structural-test-backstop` backstop).
- `CLAUDE.md`'s provenance bullet and `docs/design/provenance-model.md` should be checked for language that states or implies "the AI never invokes a sovereign-act verb itself" and updated to reflect "invokes it directly under an explicit per-invocation human approval gate" instead.
- Scope: this decision covers the *conversational, no-delegation-scope* case only. It does not change `--force`'s existing human-only enforcement, `aiwf authorize`'s delegation-scope semantics, or G-0023's separate (and still open) question of delegated `--force` within an authorized autonomous scope.
- Tracked for implementation as a gap (filed alongside this decision) so the ritual-text fix lands via a `wf-patch`.
