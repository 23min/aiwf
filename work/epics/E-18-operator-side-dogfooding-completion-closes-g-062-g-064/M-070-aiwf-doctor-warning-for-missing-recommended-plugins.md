---
id: M-070
title: aiwf doctor warning for missing recommended plugins
status: in_progress
parent: E-18
tdd: required
acs:
    - id: AC-1
      title: doctor.recommended_plugins config field accepts list
      status: open
      tdd_phase: green
    - id: AC-2
      title: doctor reads installed_plugins.json and matches project scope
      status: open
      tdd_phase: red
    - id: AC-3
      title: Each missing plugin emits one warning with install command
      status: open
      tdd_phase: red
    - id: AC-4
      title: Empty config list means no checks fire (kernel-neutral)
      status: open
      tdd_phase: red
    - id: AC-5
      title: Plugins installed for this project's scope produce no finding
      status: open
      tdd_phase: red
    - id: AC-6
      title: Plugins installed only for other scopes produce a finding
      status: open
      tdd_phase: red
    - id: AC-7
      title: doctor --self-check covers the new check end-to-end
      status: open
      tdd_phase: red
---

## Goal

Add a kernel detection mechanism for missing recommended plugins. After this milestone, `aiwf doctor` reads a new `aiwf.yaml: doctor.recommended_plugins` list and warns once per declared plugin not installed for the consumer's project scope. The kernel stays neutral on *which* plugins to recommend (empty default; consumer-declared); it just provides the detection.

This is the prerequisite for [M-071](M-071-install-ritual-plugins-in-kernel-repo-document-operator-setup-path.md): once the warning exists, M-071's fix (install the plugins, declare them in this repo's `aiwf.yaml`) can be validated by watching the warning go silent. Closes [G-062](../../gaps/G-062-aiwf-doctor-does-not-surface-missing-recommended-plugins-ritual-skills-aiwf-extensions-wf-rituals-can-be-silently-absent-from-a-consumer-repo-with-no-signal-to-operator-or-ai-assistant.md).

## Approach

Add a `doctor.recommended_plugins` field to `aiwf.yaml`'s schema (list of strings, each shaped `<name>@<marketplace>` — same format the user types into `claude /plugin install`). New doctor check `recommended-plugin-not-installed`:

1. Load the consumer repo root (already known to doctor via `--root` resolution).
2. Read `~/.claude/plugins/installed_plugins.json` (a JSON file Claude Code maintains; structure: top-level `plugins` map, each entry has scope arrays with `projectPath` and `installPath`).
3. For each entry in `doctor.recommended_plugins`, search `installed_plugins.json` for a `scope: "project"` entry whose `projectPath` matches the consumer root. Mismatch → emit one warning.

Severity: warning. Plugins are advisory; refusing on absence is too strong.

The check is read-only and fast. No file mutations, no network. Failure to read `installed_plugins.json` (file missing — Claude Code never run on this machine, or running outside Claude Code entirely) is treated as "no plugins installed" — every recommended plugin warns. That's the right behavior: if the operator can't read the file, they probably aren't running under Claude Code and the warnings are noise they can ignore by leaving the config list empty.

## Acceptance criteria

### AC-1 — doctor.recommended_plugins config field accepts list

`aiwf.yaml` accepts a `doctor.recommended_plugins` field whose value is a list of strings, each shaped `<name>@<marketplace>` (e.g., `aiwf-extensions@ai-workflow-rituals`). The field is optional; absence is equivalent to an empty list. Loaded via the existing config layer in `internal/config/`. Schema fixture in `testdata/` covers: empty list, single entry, multiple entries, missing field. Invalid shape (entry missing `@`, non-string, non-list) refuses load with a clear error pointing at the field path.

### AC-2 — doctor reads installed_plugins.json and matches project scope

`aiwf doctor` resolves `~/.claude/plugins/installed_plugins.json` (using `os.UserHomeDir()` + the documented relative path), loads the JSON, and matches each `doctor.recommended_plugins` entry against installed plugins by `(name@marketplace, projectPath==consumer-root, scope=="project")`. The match logic is exact-string for name and marketplace, exact-path for `projectPath` (after both are absolute-resolved via `filepath.Abs`). Test fixtures cover: file missing, file present with no matches, file with matches under a different `projectPath`, file with matches under the right `projectPath`.

### AC-3 — Each missing plugin emits one warning with install command

For each entry in `doctor.recommended_plugins` that doesn't match an installed plugin under the consumer's scope, doctor emits exactly one warning. The warning text includes the plugin id, the marketplace, and a one-line install hint formatted as `claude /plugin install <name>@<marketplace>`. JSON envelope output carries the same structured info (`finding.code: "recommended-plugin-not-installed"`, `finding.data: {plugin, marketplace, install_command}`). One warning per missing plugin — never deduped, never skipped. Tested with 0 / 1 / N missing plugins.

### AC-4 — Empty config list means no checks fire (kernel-neutral)

When `doctor.recommended_plugins` is absent or an empty list, the new check makes zero observations: it doesn't read `installed_plugins.json`, doesn't emit findings, and doesn't change doctor's exit code. The kernel makes no assumption about which plugins a consumer "should" have; the recommendation is consumer-declared. Tested with a fixture where the field is missing entirely and another where it's `[]`; both produce zero findings from this check.

### AC-5 — Plugins installed for this project's scope produce no finding

When all entries in `doctor.recommended_plugins` are matched by an installed plugin whose scope contains the consumer repo root, the check emits zero findings. Verified with a fixture `installed_plugins.json` whose structure mirrors the real Claude Code file (top-level `plugins` map with per-name arrays containing scope objects), where every recommended plugin has a `scope: "project"` entry whose `projectPath` resolves to the consumer root. Doctor exits 0 and produces no recommended-plugin warnings.

### AC-6 — Plugins installed only for other scopes produce a finding

The session-canonical case: a recommended plugin is installed in `installed_plugins.json` but only under a `projectPath` that does *not* match the consumer root (e.g., installed for another project). The check still fires the warning — installation under another scope is not equivalent to availability for this consumer. Tested with a fixture mirroring this session's actual state: `aiwf-extensions` installed for `/Users/x/Projects/proliminal.net`, recommended in this repo's `aiwf.yaml`. Warning fires; installation elsewhere does not silence it.

### AC-7 — doctor --self-check covers the new check end-to-end

`aiwf doctor --self-check` exercises the new check in its synthetic temp repo per the existing self-check pattern. The self-check fixture declares one entry in `doctor.recommended_plugins` (a synthetic name to avoid coupling to real marketplace state) and constructs a minimal `installed_plugins.json` in a fake `$HOME` for the subprocess; both the warning case (no matching install) and the silent case (matching install) are exercised. Per the CLAUDE.md "test the seam" rule, the integration is at the binary level: build the verb, drive it as a subprocess, assert stdout/stderr/exit code.

