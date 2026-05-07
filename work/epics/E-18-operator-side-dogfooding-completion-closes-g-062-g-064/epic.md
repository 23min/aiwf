---
id: E-18
title: Operator-side dogfooding completion (closes G-062, G-064)
status: proposed
---

## Goal

Close the operator-side gap in this repo's dogfooding of aiwf. G-038 ("kernel repo does not dogfood aiwf") landed the planning-tree migration but explicitly closed *partial* — kernel ran `aiwf init/update/check/status/import` end-to-end, but the ritual-plugin half of operator-side dogfooding was never named as a follow-up. This session surfaced the consequence: the kernel repo's design assumes `aiwf-extensions` and `wf-rituals` are present, but neither plugin is installed for this project's scope. AI assistants invoked here cannot see ritual skills (`aiwfx-start-milestone`, `wf-patch`, etc.); the standing behavior is silent ritual absence.

This epic closes the loop with two coupled milestones: [M-070](M-070-aiwf-doctor-warning-for-missing-recommended-plugins.md) adds a kernel detection mechanism (`aiwf doctor` warns on missing recommended plugins); [M-071](M-071-install-ritual-plugins-in-kernel-repo-document-operator-setup-path.md) installs the plugins in this repo, declares them in `aiwf.yaml`, and documents the install path in CLAUDE.md. M-070 ships first so M-071's fix can be validated by watching the warning go silent.

End state:
- Any consumer repo can declare `doctor.recommended_plugins` in `aiwf.yaml`; missing entries surface as `aiwf doctor` warnings.
- This repo declares its own recommended plugins, has them installed, and documents the install commands so a fresh operator (human or AI) can replicate the setup without external context.
- [G-062](../../gaps/G-062-aiwf-doctor-does-not-surface-missing-recommended-plugins-ritual-skills-aiwf-extensions-wf-rituals-can-be-silently-absent-from-a-consumer-repo-with-no-signal-to-operator-or-ai-assistant.md) and [G-064](../../gaps/G-064-kernel-repo-dogfooding-closed-partial-g-038-without-installing-the-ritual-plugins-aiwf-extensions-wf-rituals-operator-side-surface-incomplete-here-despite-framework-design-assuming-rituals-are-present.md) close.

## Scope

- M-070 — `aiwf doctor` reads `aiwf.yaml: doctor.recommended_plugins` and surfaces a warning per plugin entry that is not installed for the consumer's project scope. Detection-side; no install actions.
- M-071 — Declare this repo's recommended plugins in `aiwf.yaml`, install the plugins via `claude /plugin install`, document the setup in CLAUDE.md. Repo-side; uses M-070's check as the validation surface.

## Out of scope

- **`aiwf init` automatically installing plugins.** The kernel does not invoke Claude Code's plugin commands. The doctor check tells the operator to run the install themselves; auto-install is a different surface (Claude Code automation, not aiwf) and probably wants its own decision before implementation.
- **`aiwf doctor --install-recommended` action verb.** Same reasoning — crossing into the Claude Code surface from aiwf is a directional choice that doesn't belong inside this epic.
- **Marketplace discovery.** The kernel doesn't curate the list of recommended plugins; consumers declare what *they* want. No "default recommended set" beyond the empty list.
- **Retroactive G-038 reopen.** G-038 was knowingly closed partial; this epic is the named follow-up, not a re-litigation.
