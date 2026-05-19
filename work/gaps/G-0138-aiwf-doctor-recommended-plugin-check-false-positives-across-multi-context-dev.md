---
id: G-0138
title: 'aiwf doctor recommended-plugin check: false positives across multi-context dev'
status: open
---
## What's missing

`aiwf doctor`'s `recommended-plugins` check (`appendRecommendedPluginsReport` in `internal/cli/doctor/doctor.go`) reads the **machine-local** `~/.claude/plugins/installed_plugins.json` and asks `HasProjectScope(plugin, rootDir)` — which does an **exact-path string equality** comparison between `entry.ProjectPath` and the current `rootDir`.

This check answers *"is the plugin cached and bound to this specific working directory path?"* — a different question from *"is this plugin enabled for this project?"*. The intent-level project declaration lives in `.claude/settings.json`'s `enabledPlugins` map (project-committed, path-independent), which Claude Code's bundled binary syncs into `installed_plugins.json` via a startup routine. That sync has a skip-if-key-exists optimization: if the plugin id already has any entry in `installed_plugins.json` (for any project), it doesn't add a new entry for the current project's path.

Combined effect: a plugin can be **declared in `enabledPlugins`** for the current project AND **cached on disk** AND **functional at runtime**, but `aiwf doctor` still reports it as `recommended-plugin-not-installed` because no entry's `projectPath` matches the current rootDir.

## Why it matters

The false-positive warning bites every time the project is opened from a different mount path than the original install context:

- **Worktrees:** `~/Projects/aiwf/` (main) and `~/Projects/aiwf-foo/` (worktree) are different `projectPath`s. Plugin install for the main checkout doesn't satisfy the check for the worktree.
- **Devcontainers:** `/Users/peterbru/Projects/aiwf-devcontainer/` (host) and `/workspaces/aiwf-devcontainer/` (container) are different `projectPath`s for the same source tree. Plugins installed in one context don't satisfy the check from the other.
- **Cross-machine clones / fresh-clone path differences:** any user whose clone path differs from where the original install was recorded sees the false positive forever.

Aggravated by the plugin-index shadow-mount workaround for [claude-code#31388](https://github.com/anthropics/claude-code/issues/31388) (the devcontainer architecture from M-0132 / E-0035): the host's `~/.claude/plugins/installed_plugins.json` and the container's shadow `~/.claude-linux/plugins/installed_plugins.json` are independent files with independent entries. Even with plugins correctly installed in BOTH, each one's entries have a different `projectPath` (host vs container), so `aiwf doctor` from either side sees its own copy as "not installed for this project."

Discovered concretely during M-0132 wrap: after installing the rituals plugins at PROJECT scope inside the container, `/plugin` UI showed them as enabled, the `enabledPlugins` in `.claude/settings.json` was correct, the plugin cache was populated — but `aiwf doctor` still reported `recommended-plugin-not-installed` because the cached entry was for `/workspaces/flowtime-vnext` (from an earlier FlowTime devcontainer session sharing the same Linux shadow-mount), not `/workspaces/aiwf-devcontainer`.

## Proposed fix shape

Two candidates, in increasing thoroughness:

1. **Switch the doctor check's source of truth to `enabledPlugins` in the project's committed `.claude/settings.json`.** Replaces `pluginstate.Load(home) + HasProjectScope(plugin, rootDir)` with a markdown/JSON read of `<rootDir>/.claude/settings.json`'s `enabledPlugins` map. Path-independent (the file is in the source tree, not pinned to an absolute machine path). Version-aware only if the project pins versions in settings.json (current schema doesn't, but it could later).

2. **Cross-validate `enabledPlugins` + `installed_plugins.json`** and emit nuanced advice per state:
   - `enabledPlugins=true` + matching project-scope entry → OK (silent).
   - `enabledPlugins=true` + entry for another project / user-scope-only → "enabled but cached elsewhere; run `claude plugin install <p>@<m> --scope project` to add a binding for this path; reuses cache."
   - `enabledPlugins=true` + no cache → "enabled but not installed; `claude plugin install <p>@<m> --scope project`."
   - `enabledPlugins=false`/absent → "not declared in this project's `enabledPlugins`."

   Also corrects the current `claude /plugin install <plugin>` advice to include `--scope project` (the CLI default is user-scope per Claude Code docs — wrong for our case).

Option 1 is the simpler correctness story. Option 2 is more helpful when the operator needs to know what specifically is misconfigured. Both are kernel-side changes in `internal/cli/doctor/doctor.go` + tests.

## Related

- G-0135 / G-0136: same flavor of multi-context tax for hooks (absolute path baked at install time vs cross-environment / cross-worktree dev). Together with this gap, they form the "kernel surfaces that assume single-context dev" cluster — natural scope for a sibling milestone under E-0035 that addresses multi-context concerns holistically.
- M-0132 `## First-boot recovery` section: documents the lived symptom (Claude UI shows plugins enabled, `aiwf doctor` warns anyway) without proposing the structural fix — that's this gap.
- ADR-0006 (skills policy): adjacent to the recommended-plugins surface; the doctor's role in plugin discoverability is in scope for that ADR's conventions.

## Discipline today

When the warning fires after confirming plugins ARE enabled in `.claude/settings.json` and functional in the `/plugin` UI: ignore it; treat the warning as known-false-positive. Use Claude's own `/plugin` UI as the source of truth for "is this plugin actually working for this project."
