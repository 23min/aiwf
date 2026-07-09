---
id: G-0374
title: Worktree creation never materializes aiwf skills/agents/guidance
status: addressed
discovered_in: M-0186
addressed_by_commit:
    - 4f577230
---
## What's missing

Every ritual and instruction in this repo that creates a git worktree (`aiwfx-start-milestone`'s cut-branch step, presumably `aiwfx-start-epic`, `wf-patch`'s worktree setup, the "Default to a worktree for any branch work" instruction, and the "Subagent worktree isolation" procedure) stops at `git worktree add` and never runs `aiwf init` / `aiwf update` afterward. Since `.claude/skills/`, `.claude/agents/`, `.claude/templates/`, and `.claude/aiwf-guidance.md` are all gitignored, materialize-on-demand artifacts (ADR-0018), and `git worktree add` never checks out gitignored paths, every freshly-cut worktree starts with none of them. Nothing mechanically detects this — `aiwf doctor` reports it instantly, but nothing runs `doctor` (or `init`/`update`) automatically after a worktree is cut.

## Why it matters

This directly contradicts the project's own stated design law: "Framework correctness must not depend on LLM behavior... A guarantee that depends on the LLM remembering to invoke a skill is not a guarantee." Worktrees are the recommended default for nearly all branch work here (ADR-0023) and the mandatory mechanism for subagent isolation, so this gap sits on the critical path, not a corner case.

The blast radius varies with how a session starts:

- A session that begins in the main checkout and later `cd`s into a worktree gets lucky: Claude Code resolves the `CLAUDE.md` import tree once, at session start, so the always-on `aiwf-guidance.md` prose rides along from the main checkout even though the worktree's own copy of that file doesn't exist. But the Skill tool's "available skills" list is resolved live against the actual filesystem, so every on-demand ritual skill (`wf-vacuity`, `wf-tdd-cycle`, `wf-rethink`, `wf-review-code`, `wf-patch`, every `aiwfx-*` ritual, every `aiwf-<verb>` skill) becomes unreachable the moment the session moves into the worktree.
- A session or subagent that starts fresh with the worktree as its initial directory (the normal case for a dispatched subagent per "Subagent worktree isolation") gets neither: no always-on guidance, no invocable rituals, no error, no warning. It silently degrades to writing code without any of the TDD/vacuity/rethink/gate discipline the framework is built around.

Discovered mid-milestone (M-0186, E-0045) after `wf-rethink` failed to appear as an available skill. `aiwf doctor` confirmed 17+ missing verb skills and a completely absent `.claude/skills/`, `.claude/agents/`, `.claude/templates/` in the milestone's own worktree, despite substantial work already having happened there under the assumption that ritual discipline was live.

## Possible directions (not decided)

- Mechanical, strongest: an `aiwf`-provided wrapper (e.g. `aiwf worktree add`) that does `git worktree add` + `aiwf init`/`update` atomically, so a bare worktree without materialized artifacts becomes structurally impossible wherever it's adopted.
- Mechanical, lighter: a session-start or directory-change hook that checks for `.claude/skills/` presence whenever cwd is under `.claude/worktrees/` and refuses/warns before proceeding.
- Cheapest, weakest: add an explicit "run `aiwf update`" step to every ritual/instruction that cuts a worktree — still LLM-memory-dependent, the exact failure mode that produced this gap.
