---
id: G-0381
title: aiwfx-start-epic still hands off sovereign-act promote instead of gating
status: open
---
## What's missing

Per D-0032: `aiwfx-start-epic`'s `SKILL.md` currently instructs the AI assistant to hand `aiwf promote E-NN active` off to the human to type themselves — "the operator is human; an AI assistant orchestrating the conversation does not invoke the verb itself" (step 6 prose), echoed in the skill's "Anti-patterns" ("Letting an AI assistant run `aiwf promote E-NN active` directly") and "Constraints" sections. D-0032 reverses this: the assistant should present the exact command as an explicit approve/deny gate and, on approval, execute it directly (no `--actor` override — the bare command already resolves to the human's own git identity and passes the kernel's sovereign-act gate).

The ritual text has not been rewritten yet. Scope of the fix:

- Rewrite `aiwfx-start-epic`'s step 6 promotion prose, its "Anti-patterns" entry, and its "Constraints" bullet to describe the approve/deny-gate-then-execute shape instead of the hand-off shape.
- Add the companion structural test under `internal/policies/` this repo's `skill-edit-structural-test-backstop` convention requires for any `embedded-rituals/**/SKILL.md` edit.
- Check `CLAUDE.md`'s provenance bullet and `docs/pocv3/design/provenance-model.md` for language that states or implies the AI never invokes a sovereign-act verb itself, and update anything found to match D-0032's decision.

## Why it matters

Until this lands, the shipped ritual still tells every AI assistant running `aiwfx-start-epic` to refuse the verb and redirect the human to a shell — the exact friction D-0032 was recorded to remove. The decision is accepted but inert without this follow-through: a consumer repo running `aiwf update` today still receives the old, reversed guidance.
