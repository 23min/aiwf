---
id: M-0135
title: 'aiwf doctor containerized-env awareness: detection + mount check'
status: in_progress
parent: E-0035
tdd: required
acs:
    - id: AC-1
      title: Container detection one-liner in aiwf doctor output
      status: open
      tdd_phase: green
    - id: AC-2
      title: Shadow-mount status check in aiwf doctor (in container only)
      status: open
      tdd_phase: red
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

### AC-1 — Container detection one-liner in aiwf doctor output

**Pass criterion**: `aiwf doctor` output contains an `env:` line
near the top of the report (sibling of `binary:`, `config:`,
`actor:`). The value is `devcontainer` (with the detected signal in
parentheses) when running in a container, or `host` when not. A new
`InContainer()` probe in `internal/cli/doctor/env.go` checks
`/.dockerenv` file existence and the `AIWF_DEVCONTAINER` env var
(value must be a non-empty truthy literal — `1`, `true`,
case-insensitive). The line is informational only — never
increments problems.

**Edge cases**: both signals present → `devcontainer (/.dockerenv +
AIWF_DEVCONTAINER)`; only `/.dockerenv` (e.g., non-devcontainer
Docker shell) → `devcontainer (/.dockerenv)`; only
`AIWF_DEVCONTAINER=1` (e.g., env-var leak, no Docker) →
`devcontainer (AIWF_DEVCONTAINER)`; neither → `host`. Empty or
non-truthy `AIWF_DEVCONTAINER` value (e.g., `AIWF_DEVCONTAINER=0`
or `AIWF_DEVCONTAINER=`) is treated as absent. Symlinked
`/.dockerenv` to a real file still counts (stat, not lstat).

**Code references**: `internal/cli/doctor/env.go` (new file —
`InContainer()` probe + the small helper for output formatting);
`internal/cli/doctor/doctor.go` `DoctorReport` (call site —
prepend the `env:` line near the existing `binary:` /
`config:` block); regression test in
`internal/cli/integration/doctor_cmd_test.go` exercises a
table-driven fixture set covering the four signal combinations
above via `t.Setenv` (test cannot `t.Parallel` because env mutation
is process-global; serial test, no parallelism cost since only one
test in the table).

### AC-2 — Shadow-mount status check in aiwf doctor (in container only)

**Pass criterion**: when `InContainer()` returns true, `aiwf
doctor` output contains a `plugin-index-mount:` line indicating
the state of the bind-mount target `<userHome>/.claude/plugins/`
(the in-container path that the host's `~/.claude-linux/plugins`
bind-mounts onto per `.devcontainer/devcontainer.json`'s mount
entry). Three observable states: `ok (<N> plugin entries cached)`
(directory exists, contains at least one subentry); `empty (mount
target exists but no plugin entries — first rebuild before
initialize.sh, or shadow-mount not yet seeded)`; `missing (mount
target does not exist — devcontainer.json mount entry stripped or
container rebuild failed mid-postcreate)`. When `InContainer()`
returns false, the `plugin-index-mount:` line is omitted entirely
(the check has no value on host; the shadow-mount workaround only
applies inside the container). The line is read-only and never
increments problems.

**Edge cases**: `<home>/.claude/plugins/` is a regular file rather
than a directory → reported as `missing` (target shape is wrong);
`os.UserHomeDir()` returns an error → propagated to a single
`plugin-index-mount: <err>` line, problems unchanged; the directory
contains hidden entries (`.lock`, `.tmp`) — count only top-level
entries that aren't hidden (the count is operator-facing, not
forensic); plugin count is large (>100) → render as
`ok (100+ plugin entries cached)` to avoid runaway formatting on
unusual setups (cap is opinionated but matches typical cache
sizes).

**Code references**: `internal/cli/doctor/env.go` (alongside the
AC-1 helpers — new `shadowMountStatus(home string) (state, count,
err)` probe); `internal/cli/doctor/doctor.go` `DoctorReport`
gated on `InContainer()` from AC-1; regression test in
`internal/cli/integration/doctor_cmd_test.go` covers the four
states via fixture trees (in-container + dir-with-entries,
in-container + empty-dir, in-container + missing-dir,
not-in-container — line absent). Tests use `t.Setenv` for the
`AIWF_DEVCONTAINER` flag to control `InContainer()` deterministically;
serial (no t.Parallel) because env mutation is process-global.

