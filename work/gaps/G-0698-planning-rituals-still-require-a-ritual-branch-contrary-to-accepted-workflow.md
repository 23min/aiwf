---
id: G-0698
title: Planning rituals still require a ritual branch contrary to accepted workflow
status: open
discovered_in: M-0341
---
## What's missing

The embedded `aiwfx-plan-epic` and `aiwfx-plan-milestones` skills instruct the operator to merge planning from a ritual branch and switch to main. Accepted D-0073 requires planning on main with no ritual branch or merge step. The contradiction is in `internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md` under Closing the planning session and `aiwfx-plan-milestones/SKILL.md` step 10.

Measured at commit `d1f03bc04` from the repository root. The expected instructions agree with the accepted decision; the following read-only commands show the conflicting text:

```sh
rg -H -n -o 'There is no ritual branch and no merge step\.' work/decisions/D-0073-planning-happens-on-main-implementation-work-happens-in-a-worktree.md
rg --sort path -n -o 'git checkout main' internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md
```

Observed output:

```text
work/decisions/D-0073-planning-happens-on-main-implementation-work-happens-in-a-worktree.md:26:There is no ritual branch and no merge step.
internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-epic/SKILL.md:107:git checkout main
internal/skills/embedded-rituals/plugins/aiwf-extensions/skills/aiwfx-plan-milestones/SKILL.md:119:git checkout main
```

The surrounding paragraphs explicitly request a merge; neither ritual creates the branch it assumes. This reproduces an instruction contradiction, not an observed execution by a live assistant.

## Why it matters

An assistant following the shipped skills is directed to a planning workflow that conflicts with the accepted decision. It can request an unnecessary merge or try to check out main from an implementation worktree where main is already held elsewhere. The same shared planning text is rendered for Claude and Codex, so switching hosts does not resolve the conflicting instructions.
