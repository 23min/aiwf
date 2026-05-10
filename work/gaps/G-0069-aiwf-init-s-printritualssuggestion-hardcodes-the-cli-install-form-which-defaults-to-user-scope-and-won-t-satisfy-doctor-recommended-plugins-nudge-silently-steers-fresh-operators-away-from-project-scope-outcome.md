---
id: G-0069
title: aiwf init's printRitualsSuggestion hardcodes the CLI install form, which defaults to user scope and won't satisfy doctor.recommended_plugins; nudge silently steers fresh operators away from project-scope outcome
status: open
discovered_in: M-0070
---

## What's missing

A coherent install nudge from `aiwf init` that steers operators to the install path that actually satisfies M-0070's `doctor.recommended_plugins` check.

`cmd/aiwf/rituals.go`'s `printRitualsSuggestion` (called from `aiwf init`'s success path) tells operators to run:

```
/plugin marketplace add 23min/ai-workflow-rituals
/plugin install aiwf-extensions@ai-workflow-rituals
```

The CLI form `/plugin install <name>@<marketplace>` defaults to **user** scope. When the plugin is already user-installed (the common case for any operator who's used the rituals on another repo), it short-circuits with "already installed globally" and produces no project-scope entry in `~/.claude/plugins/installed_plugins.json`. M-0070's check then continues to warn — the operator followed the nudge but the warning won't go silent.

Two candidate fixes:
- Rewrite the nudge to direct operators to the interactive `/plugin` menu (Discover tab → Project scope), per the procedure now documented in [`CLAUDE.md`](../../CLAUDE.md)'s "Operator setup" section.
- Or read `aiwf.yaml: doctor.recommended_plugins` and emit nudge text that names the consumer's specific plugin set rather than the hardcoded `aiwf-extensions` example.

The choice is a directional decision (do we ever cross from `aiwf` into Claude Code's plugin surface beyond text suggestions?) — see E-0018's *Out of scope* notes.

## Why it matters

The whole point of E-0018 was closing the discoverability loop: an operator runs `aiwf init`, sees a nudge, follows it, ends up with a working setup. With the current nudge text, step 3 fails silently — the install lands in user scope instead of project scope, M-0070's check continues to warn, and the operator either grinds against an opaque warning or learns to ignore it. Both outcomes erode the framework's trust in its own signals.

This was discovered live during M-0071/AC-2 in the same session that built M-0070 — the operator (a human + AI pair) hit the friction immediately and worked around it via the interactive `/plugin` menu. The lived experience is the forcing function; the documented workaround in CLAUDE.md is the bandage. The fix removes the bandage.
