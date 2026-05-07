---
id: M-071
title: Install ritual plugins in kernel repo + document operator setup path
status: in_progress
parent: E-18
tdd: required
acs:
    - id: AC-1
      title: aiwf.yaml declares aiwf-extensions and wf-rituals as recommended
      status: met
      tdd_phase: done
    - id: AC-2
      title: Both plugins are installed for this repo's project scope
      status: met
      tdd_phase: done
    - id: AC-3
      title: M-070's doctor warning goes silent after install (validates G-062)
      status: met
      tdd_phase: done
    - id: AC-4
      title: CLAUDE.md gains a setup section listing recommended plugins
      status: met
      tdd_phase: done
---

## Goal

Bring this repo's operator-side dogfooding to a complete state. After this milestone:

- This repo's `aiwf.yaml` declares the ritual plugins it expects operators to have, using the new field from [M-070](M-070-aiwf-doctor-warning-for-missing-recommended-plugins.md).
- Both `aiwf-extensions@ai-workflow-rituals` and `wf-rituals@ai-workflow-rituals` are installed for this repo's project scope.
- `aiwf doctor` runs cleanly (no recommended-plugin warnings).
- CLAUDE.md tells a fresh operator (human or AI) the install commands and a one-line rationale, so the setup is reproducible without external context.

Closes [G-064](../../gaps/G-064-kernel-repo-dogfooding-closed-partial-g-038-without-installing-the-ritual-plugins-aiwf-extensions-wf-rituals-operator-side-surface-incomplete-here-despite-framework-design-assuming-rituals-are-present.md). The validation surface is M-070's check — when this milestone's commits land, the warnings M-070 introduces should go silent in this repo.

## Approach

Three small changes, mostly mechanical, in the order:

1. **Declare in `aiwf.yaml`** — add `doctor.recommended_plugins: ["aiwf-extensions@ai-workflow-rituals", "wf-rituals@ai-workflow-rituals"]`. Single config edit; one commit.
2. **Install via `claude /plugin install`** — run the install command for each plugin, scoped to this project. This is an operator action (humans run `claude` slash commands; AI assistants in Claude Code can suggest them via `! claude /plugin install ...` in the prompt). Verifiable in `~/.claude/plugins/installed_plugins.json` after install.
3. **Document in CLAUDE.md** — add a "Recommended plugins" subsection naming the plugins, the install commands, and a one-line rationale ("the framework's planning rituals live here; install them or `aiwf doctor` will warn"). Cross-reference this milestone and the recommended-plugins config field.

The plugin installs are not tracked in git (plugins live under `~/.claude/`, outside the repo), so AC-2's verification is by reading `installed_plugins.json` post-install rather than by checking commits. AC-3 is the round-trip validator: M-070's warning should be present before AC-2's install and absent after.

## Acceptance criteria

### AC-1 — aiwf.yaml declares aiwf-extensions and wf-rituals as recommended

The repo's `aiwf.yaml` gains a `doctor.recommended_plugins` field whose value is the two-element list `["aiwf-extensions@ai-workflow-rituals", "wf-rituals@ai-workflow-rituals"]`. The change is one git commit. Format follows whatever schema M-070 establishes; if M-070's schema-fixture tests pass with this declaration, the AC is met for the structural side. After this commit, `aiwf doctor` (run with M-070's check active) should warn about both plugins until AC-2 installs them.

### AC-2 — Both plugins are installed for this repo's project scope

`~/.claude/plugins/installed_plugins.json` contains entries for both `aiwf-extensions@ai-workflow-rituals` and `wf-rituals@ai-workflow-rituals` with `scope: "project"` and `projectPath` matching this repo's absolute root. Verified by reading the file directly (no aiwf or git involvement; this is a Claude Code state assertion). The install is performed by running `claude /plugin install aiwf-extensions@ai-workflow-rituals` and `claude /plugin install wf-rituals@ai-workflow-rituals`; the operator (human or AI suggesting via `! ...`) drives this step.

### AC-3 — M-070's doctor warning goes silent after install (validates G-062)

After AC-1 (declaration) and AC-2 (install) both land, `aiwf doctor` produces zero `recommended-plugin-not-installed` warnings against this repo. This is the round-trip validation that M-070's detection logic works: warning fires when state is wrong, goes silent when state is right. Manual verification is sufficient (run doctor before and after install; observe transition); a CI assertion would be brittle since the local plugin-install state isn't controlled by CI.

### AC-4 — CLAUDE.md gains a setup section listing recommended plugins

CLAUDE.md gains a new section (under "Working on the PoC" or a new top-level "Operator setup") that:
- Names the two recommended plugins and their marketplace.
- Provides the exact install commands.
- Gives a one-line rationale ("the framework's planning rituals live here; install them or `aiwf doctor` will warn").
- Cross-references this milestone (M-071) and the M-070 doctor check.
- Names this section as the answer to "what should a new operator do after cloning?" — closing the discoverability loop the gap chain (G-038 → G-064) opened.

The section is short — five to ten lines is appropriate. The point is not to re-explain the rituals (that's the plugins' own SKILL.md files); it's to *signal that the plugins exist and are expected*.

