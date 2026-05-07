---
id: G-062
title: aiwf doctor does not surface missing recommended plugins; ritual skills (aiwf-extensions, wf-rituals) can be silently absent from a consumer repo with no signal to operator or AI assistant
status: open
---

## What's missing

`aiwf doctor` performs drift checks on the binary version, hook freshness, id collisions, and `--self-check` end-to-end coverage. It says nothing about Claude Code plugin installation state, even though several of the framework's load-bearing rituals live in plugin skills shipped through a marketplace (e.g., `ai-workflow-rituals` providing `aiwf-extensions` and `wf-rituals`).

Concretely: when a consumer repo expects rituals like `aiwfx-start-milestone` (preflight + branch + iterate), `aiwfx-wrap-milestone`, or `wf-patch` to be available to its operators (human or AI), and those plugins happen to be installed only for *another* project's scope (or not at all), `aiwf doctor --self-check` exits clean. The operator gets no signal that the ritual surface is silently absent. AI assistants in particular have no way to know a ritual exists if its skill isn't present in their available-skills list — and the kernel doesn't tell them either.

A doctor finding (warning severity) per missing plugin would close the loop:

```
warning: recommended plugin 'aiwf-extensions@ai-workflow-rituals' is not installed for this project
  rituals affected: aiwfx-plan-epic, aiwfx-plan-milestones, aiwfx-start-milestone, aiwfx-wrap-milestone, ...
  install: claude /plugin install aiwf-extensions@ai-workflow-rituals
```

## Why it matters

This turn was a real instance: the operator (Claude in this session) had no way to know that `aiwfx-start-milestone` exists, that it carries the branch-setup ritual, or that the absence of a "start epic" ritual is by design (the chain collapses epic-start into first-milestone-start). I bypassed the ritual entirely — promoted E-17 to active on `poc/aiwf-v3` without a branch — because nothing in my skill list told me a ritual existed. The user had to know about the plugins independently and prompt me. That's a discoverability failure squarely in the "kernel functionality must be AI-discoverable" principle's territory, even though the ritual itself is plugin-side: the *gap* between "kernel knows the plugin is recommended" and "kernel reports its absence" is the kernel's responsibility, because the kernel owns `aiwf doctor`.

Without this signal:

- Operators silently work without the rituals their project's planning model assumes.
- Branch-discipline gaps (G-059) and patch-ritual gaps (G-060) appear as kernel-design bugs when in fact the rituals exist at the skill layer — they're just not installed *here*.
- Each new AI session re-discovers the absence by friction, never by signal.

## Suggested shape

- New `aiwf.yaml` field: `doctor.recommended_plugins: [<name>@<marketplace>, ...]` — consumer-declared, kernel-neutral. Empty-by-default so kernel does not pin a marketplace.
- `aiwf doctor` reads `~/.claude/plugins/installed_plugins.json`, matches `scope: "project"` entries against the consumer's repo root, and emits one warning per missing entry from the configured list.
- Severity: warning. Plugins are advisory; refusal is too strong.
- The `--self-check` path includes a fixture run with a `recommended_plugins` declaration to ensure the new check is exercised end-to-end, not just unit-tested.

A future enhancement (out of scope for this gap) could add a `aiwf doctor --install-recommended` action that emits `claude /plugin install …` lines or shells them out, but that crosses into the Claude Code surface and probably wants its own decision before implementation.

