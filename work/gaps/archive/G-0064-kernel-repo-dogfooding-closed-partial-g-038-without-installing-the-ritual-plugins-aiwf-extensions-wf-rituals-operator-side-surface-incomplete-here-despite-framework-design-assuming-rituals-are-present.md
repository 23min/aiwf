---
id: G-0064
title: Kernel repo dogfooding closed partial (G-0038) without installing the ritual plugins (aiwf-extensions, wf-rituals); operator-side surface incomplete here despite framework design assuming rituals are present
status: addressed
addressed_by:
    - M-0071
---

## What's missing

### History

G-0038 ("the kernel repo does not dogfood aiwf — feasibility and fit need investigation") was closed on 2026-05-05 with the rationale:

> *"G38 partial dogfood landed: gaps + poc-plan migrated, kernel runs aiwf init/update/check/status/import end-to-end. Forward gaps go through aiwf add gap. Open follow-ups: G-0048 (core.hooksPath), G-0049 (legacy gap-resolved-has-resolver noise)."*

Two follow-ups were named (G-0048, G-0049). Both are kernel-side. **A third follow-up was missed**: the ritual plugins (`aiwf-extensions`, `wf-rituals` from the `ai-workflow-rituals` marketplace) — which the framework's own design assumes are present in any consumer repo using the planning lifecycle — were never installed in this repo. They are installed only for `/Users/peterbru/Projects/proliminal.net` per `~/.claude/plugins/installed_plugins.json`.

### Concrete defects in this repo

1. **Plugins not installed.** No `aiwf-extensions` or `wf-rituals` entries with `scope: "project"` and `projectPath` matching this repo. AI assistants invoked here do not see ritual skills (`aiwfx-plan-epic`, `aiwfx-start-milestone`, `aiwfx-wrap-milestone`, `wf-patch`, etc.) in their available-skills list.
2. **No documented install path.** Neither CLAUDE.md, the `aiwf init` flow, nor any design doc names the plugins as part of consumer setup. A new operator (human or AI) starting on this repo has no signal that the plugins exist or are recommended.
3. **Real-world consequence.** This turn (2026-05-07): Claude promoted E-0017 to `active` on `poc/aiwf-v3` with no preflight, no body check, no branch creation, no authorization decision — because no ritual skill was present to suggest those steps. The user had to know about the plugins from outside this repo to surface the gap.
4. **No detection.** Even when `aiwf doctor` is run, it says nothing about plugin state — that's a separate, prerequisite gap (see Dependencies below).

### Dependencies and ordering

**This gap depends on G-0062 landing first.** G-0062 (`aiwf doctor` surfaces missing recommended plugins) is the detection mechanism. Closing G-0064 by installing plugins + documenting the install path is much more validatable when there's a `doctor` warning that goes silent on install. Without G-0062, the only way to verify G-0064's fix is by hand-inspecting `~/.claude/plugins/installed_plugins.json` — fine for a one-off but not auditable.

Suggested order:

1. **G-0062 first** — implement the doctor check, ship the `doctor.recommended_plugins` config field, see the warning fire on this repo.
2. **G-0064 second** — declare the recommended plugins in this repo's `aiwf.yaml`, install them, see the warning go silent. Document the install path in CLAUDE.md (and possibly in `aiwf init`'s output for new consumer repos).

These two probably want to live under one epic ("complete operator-side dogfooding" or similar), with G-0062's milestone first and G-0064's milestone second.

### Suggested shape for the G-0064 fix

- Add `doctor.recommended_plugins: ["aiwf-extensions@ai-workflow-rituals", "wf-rituals@ai-workflow-rituals"]` to this repo's `aiwf.yaml`.
- Run `claude /plugin install aiwf-extensions@ai-workflow-rituals` and `claude /plugin install wf-rituals@ai-workflow-rituals`.
- Update CLAUDE.md to name the plugins as part of consumer setup, with the install commands and a one-line rationale ("the framework's planning rituals live here; install them or `aiwf doctor` will warn").
- Optional follow-up: `aiwf init` could print install hints when the kernel-known recommended plugins are absent in the consumer environment. Out of scope for the gap's first close — that's a future ergonomic improvement.

## Why it matters

- **Dogfooding is load-bearing for the framework's credibility.** A framework that doesn't use itself in its own development surfaces drift later, in consumers. G-0038 was knowingly closed partial; the un-named follow-up is how the partial state went silently uncorrected.
- **Every AI session in this repo re-discovers the absence by friction.** This is the inverse of the kernel's "AI-discoverability" principle: a load-bearing capability (the rituals) is undocumented and unsignaled in the framework's own repo.
- **G-0059 and G-0060 (branch model, patch ritual) appear as unsolved kernel-design gaps when in fact the rituals exist at the skill layer** — they're just not installed *here*. Closing G-0064 collapses part of those gaps into "skill present, rule advisory."
- **The G-0064 fix is the cleanest validation of G-0062.** The doctor check fires on the partial state; install resolves it; the warning goes silent. That's the kind of round-trip that closes a gap convincingly.

## Predecessor / sibling references

- **G-0038** (addressed) — predecessor: kernel-side dogfooding, closed partial.
- **G-0062** (open) — sibling, prerequisite: doctor surfaces missing plugins.
- **G-0063** (open) — sibling: start-epic ritual not defined in framework design (different surface — design vs. installation).
- **G-0059, G-0060** (open) — partially collapsible once rituals are installed (skill-layer answers exist for both).

