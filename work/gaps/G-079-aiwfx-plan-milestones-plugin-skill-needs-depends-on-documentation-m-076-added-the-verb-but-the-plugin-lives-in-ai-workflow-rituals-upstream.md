---
id: G-079
title: aiwfx-plan-milestones plugin skill needs --depends-on documentation; M-076 added the verb but the plugin lives in ai-workflow-rituals upstream
status: open
discovered_in: M-076
---

## What's missing

The `aiwfx-plan-milestones` skill in the [ai-workflow-rituals](https://github.com/23min/ai-workflow-rituals) plugin still says "edit M-NNN's frontmatter" for milestone dependency declaration. M-076 shipped two writer verbs that obviate the hand-edit:

- `aiwf add milestone --depends-on M-PPP[,M-QQQ]` for allocation-time edges
- `aiwf milestone depends-on M-NNN --on M-PPP[,M-QQQ] [--clear]` for post-allocation edits

The plugin skill should be updated to invoke the verb instead of pointing at frontmatter. Without this update, AI assistants invoked under the planning ritual continue to hand-edit `depends_on:` and trip the kernel's `provenance-untrailered-entity-commit` warning — same friction G-072 was supposed to close.

## Why it matters

The kernel's "AI-discoverability" principle (CLAUDE.md) requires that every kernel verb is reachable through channels an AI assistant routinely consults — including the rituals plugin's planning skill. M-076's `aiwf-add` skill update covers in-tree consumers, but the rituals-plugin consumer (which is what most operators install) still teaches the wrong path. Filing this gap so the upstream PR is tracked; closes when the plugin is updated and re-released.

The plugin lives in a separate repo (`23min/ai-workflow-rituals`), so the fix is an upstream PR, not an in-tree edit. M-076's spec called this out as an AC-6 deferral; this gap is the receiving artifact.
