---
id: M-071
title: Install ritual plugins in kernel repo + document operator setup path
status: done
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

## Work log

The kernel records the per-AC red→green→done→met timeline; `aiwf history M-071/AC-<N>` is the authoritative chronology. This section captures the per-AC outcome only.

### AC-1 — declare both plugins in aiwf.yaml

Added a `doctor:` block to the consumer's `aiwf.yaml` listing both `aiwf-extensions@ai-workflow-rituals` and `wf-rituals@ai-workflow-rituals` under `recommended_plugins`. The block carries a one-paragraph comment explaining the field's purpose and pointing forward at `CLAUDE.md`'s operator-setup section. After this change, M-070's check fired for `wf-rituals` (the plugin not yet installed at project scope) and stayed silent for `aiwf-extensions` (already installed) — exactly the round-trip M-071 was designed to validate.

### AC-2 — install both plugins at project scope

Both plugins now have project-scope entries in `~/.claude/plugins/installed_plugins.json` with `projectPath: /Users/peterbru/Projects/ai-workflow-v2`. The user's `.claude/settings.json` gained a corresponding `enabledPlugins` entry for `wf-rituals` (alongside the pre-existing `aiwf-extensions` entry from the M-070 chore commit). The install was performed via the interactive `/plugin` menu — see *Decisions* for why the CLI form `claude /plugin install <name>@<marketplace>` is not equivalent.

### AC-3 — doctor warning silent after install

Live `aiwf doctor` against this repo produces zero `recommended-plugin-not-installed:` lines. Validates the M-070 check end-to-end against real consumer state: the warning fires when state is wrong (AC-1 baseline), goes silent when state is right (AC-2 outcome). M-070 wins its dogfooding moment.

### AC-4 — CLAUDE.md "Operator setup" section

Added a new top-level `## Operator setup` section between "How to validate changes" and "Working on the PoC". Six lines of prose plus one fenced command block. Calls out the project-scope-vs-user-scope distinction explicitly (the CLI form short-circuits to user scope; only the interactive menu offers a project-scope choice — see *Decisions*). Cross-references aiwf.yaml's `doctor.recommended_plugins`, M-070, M-071, G-064, and E-18.

## Decisions made during implementation

No new ADR / D-NNN was opened during M-071. One decision is worth recording inline because it shapes the operator-facing prose:

- **CLAUDE.md documents the interactive `/plugin` menu, not the CLI shorthand.** The CLI form `claude /plugin install <name>@<marketplace>` defaults to **user** scope (and short-circuits with "already installed globally" if the plugin exists at any scope). Only the interactive `/plugin` menu — Discover tab → select plugin → choose Project scope — produces a project-scope entry in `installed_plugins.json` that satisfies M-070's check. The setup docs say so explicitly. This nuance was discovered live during AC-2; documenting it inline was the cheaper option than carrying it as a follow-up gap.

## Validation

```
$ /tmp/aiwf show M-071                  → 4/4 ACs met, all phases done
$ /tmp/aiwf check                       → 0 errors, 4 warnings (all pre-existing,
                                          none on M-071)
$ /tmp/aiwf doctor                      → zero recommended-plugin-not-installed
                                          lines (live round-trip)
$ go build -o /tmp/aiwf ./cmd/aiwf      → exit 0
$ go vet ./...                          → clean
$ golangci-lint run --timeout=5m ./...  → 0 issues
$ wf-doc-lint scoped                    → 0 findings
```

No Go test suite re-run was performed because M-071 touched zero lines of Go code. The diff is `aiwf.yaml` (+9), `.claude/settings.json` (+1), `CLAUDE.md` (+15) — no behavioral change to test against.

Branch-coverage audit: trivially N/A — no code changes.

## Deferrals

No work deferred from this milestone. One follow-up surfaced during AC-2 that was *named* in M-070's wrap as well, now reinforced:

- **`aiwf init`'s `printRitualsSuggestion` hardcodes the CLI install form** (`/plugin install aiwf-extensions@ai-workflow-rituals`) even though that form defaults to user scope and short-circuits when the plugin is globally installed. The nudge as written silently steers fresh operators away from the project-scope outcome the framework actually wants. M-070's wrap flagged this as a follow-up; M-071's setup-flow experience confirms the friction is real. Migrating the nudge to either (a) reference the interactive menu or (b) read `aiwf.yaml: doctor.recommended_plugins` is a separate decision worth its own ticket. Not opened as a gap here — leaving it under M-070's wrap *Deferrals* list to avoid duplicating the same item across milestones.

## Reviewer notes

- **Why .claude/settings.json is in the diff.** The interactive `/plugin` install at project scope writes both `installed_plugins.json` (Claude Code state) AND `.claude/settings.json` (project-shared `enabledPlugins`). The settings.json change is the natural artifact of AC-2 — it's the user-shareable record that lets a teammate cloning the repo see "this project enables these plugins." Bundling it in M-071's wrap commit keeps the AC-2 evidence in one place.
- **Why no separate gap was opened for the `printRitualsSuggestion` issue.** It's already on file under M-070's *Deferrals* (mentioned at M-070 wrap). Opening a new gap would create two equally-canonical places pointing at the same TODO. Single-source-of-truth wins.
- **Why CLAUDE.md's section calls out the CLI vs. menu distinction so explicitly.** It's the exact failure mode the operator (any operator) will hit on their first attempt. M-071's AC-2 hit it twice in this very session. A doc that just says "run the install command" without naming the trap shifts the cost from "documented surprise" to "rediscovered surprise" — the latter compounds.
- **The `aiwfx-*` and `wf-*` skills disappear from the available-skills list mid-wrap.** The user uninstalled all plugins to start fresh for the project-scope install; during that window this assistant temporarily lost access to `aiwfx-wrap-milestone` and friends. The skills came back after `/reload-plugins` completed AC-2. Worth being aware of: any operator who uninstalls plugins mid-flow will see this exact UX, and it's not a bug — it's the visible consequence of the kernel-vs-plugin separation. The kernel `aiwf-*` skills are embedded in the repo and stayed available throughout.

