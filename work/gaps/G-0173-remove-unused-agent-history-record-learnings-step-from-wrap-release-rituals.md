---
id: G-0173
title: Remove unused agent-history Record-learnings step from wrap/release rituals
status: open
---
## What's missing

Three aiwf-extensions rituals end with a "Record learnings → `work/agent-history/<agent>.md`" step that is, in practice, never used — no operator or agent reads or maintains the `work/agent-history/` files, so the step is dead ceremony that pads every wrap/release with a write nobody consumes.

## Scope (rituals plugin repo — `ai-workflow-rituals`)

The step appears in three `SKILL.md` files under `plugins/aiwf-extensions/skills/`:

- `aiwfx-wrap-milestone/SKILL.md` — step 12 "Record learnings" (→ `work/agent-history/builder.md`)
- `aiwfx-wrap-epic/SKILL.md` — step 12 "Record learnings" (→ `work/agent-history/<agent>.md`)
- `aiwfx-release/SKILL.md` — the agent-history record step

`wf-patch`'s step 10 is a generic "Reflection — record where the project records such things" and does **not** hardcode agent-history, so it can stay as-is (or be left to the operator's judgment).

## Proposed change

Remove the "Record learnings" / agent-history step from the three skills above, renumbering the surrounding steps. The change is markdown-only — the rituals repo stays pure (no tests there, per the cross-repo plugin-testing convention).

Open question: whether to also delete the existing `work/agent-history/` directory + files in consumer repos (this repo has one), or just stop the rituals from writing new entries. Leaning: stop writing new entries now; sweep existing files as separate cleanup if they're confirmed unused.

## Why this is tracked in the aiwf repo

The change lands in the upstream rituals plugin repo, but this repo's planning tree is where workflow-ritual concerns are tracked (precedent: the plugin-install gaps G-064/M-071). The resolving commit will live in `ai-workflow-rituals`, not here.
