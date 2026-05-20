---
id: M-0135
title: 'aiwf doctor containerized-env awareness: detection + mount check'
status: draft
parent: E-0035
tdd: required
---
## Goal

After M-0132 + M-0133 fixed the dominant container-specific friction
(plugin false-positive warnings, hook tug-of-war), the residual
container concerns in `aiwf doctor` are surface-level: operators
(and LLMs reading doctor output) lack a quick "where am I"
environment signal, and the shadow-mount workaround for
[claude-code#31388](https://github.com/anthropics/claude-code/issues/31388)
has no automated health check. Add a small env-awareness pass to
`aiwf doctor`: one line indicating container vs host detection,
plus a shadow-mount target sanity check when in container.

## Approach

Two coordinated additions to `internal/cli/doctor/doctor.go`,
each carrying its own AC with a mechanical assertion.

1. **Container detection one-liner.** Add an `InContainer()` probe
   in the doctor package (`internal/cli/doctor/env.go` — local to
   the only caller per the kernel's "no abstraction without a
   second consumer" rule). Detection signals: `/.dockerenv` file
   exists (Docker convention) and/or `AIWF_DEVCONTAINER=1` env
   var (set by `.devcontainer/devcontainer.json`'s `containerEnv`).
   Emit one line near the top of doctor output, e.g.,
   `env:       devcontainer (/.dockerenv)` or
   `env:       devcontainer (/.dockerenv + AIWF_DEVCONTAINER)` or
   `env:       host`. The line is informational; never increments
   problems.

2. **Shadow-mount status check.** When `InContainer()` returns
   true, probe the bind-mount target — `<home>/.claude/plugins/`
   (the in-container path that backs onto `~/.claude-linux/plugins`
   on the host per `.devcontainer/devcontainer.json`'s mount
   entry). Emit a `plugin-index-mount:` line with one of:
   `ok (<N> plugin entries cached)`, `empty (target exists but
   no plugin entries — first rebuild before `initialize.sh`?)`,
   `missing (target dir absent — mount not configured)`. When
   not in container, omit the line entirely (no value on host;
   the shadow-mount only matters inside the container). Like
   the env line, never increments problems (operational
   diagnostic, not a hard gate).

**Deliberately out of scope (deferred):**
- Container-aware advice strings on existing doctor messages
  (swapping "run `aiwf init`" for "rebuild the container" in
  select contexts). The advice changes are subjective and would
  feel preachy; the bigger value is environment surfacing, not
  prescriptive direction. File a follow-up gap if friction
  surfaces in practice.
- Deeper plugin-index introspection (validating specific
  `installed_plugins.json` contents, cross-checking enabledPlugins
  per-marketplace, etc.). M-0133 AC-3 already handled the
  source-of-truth question; doctor doesn't need to re-litigate
  it here. The mount-status check stays shallow on purpose.

## Acceptance criteria

ACs land via `aiwf add ac M-NNNN`.
