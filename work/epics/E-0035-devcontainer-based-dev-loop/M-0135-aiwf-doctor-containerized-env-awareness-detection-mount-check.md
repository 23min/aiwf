---
id: M-0135
title: 'aiwf doctor containerized-env awareness: detection + mount check'
status: in_progress
parent: E-0035
tdd: required
acs:
    - id: AC-1
      title: Container detection one-liner in aiwf doctor output
      status: met
      tdd_phase: done
    - id: AC-2
      title: Shadow-mount status check in aiwf doctor (in container only)
      status: met
      tdd_phase: done
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

## Work log

### AC-1 — Container detection one-liner

- Red→green bundled (`252173e6`): `internal/cli/doctor/env.go`
  (`InContainer()` + `detectContainer()` + `isTruthy()` helpers)
  plus the `env:` line wired into `DoctorReport`. Test set:
  `TestDetectContainer` (7-row table over signal combinations +
  truthy/falsy edge cases) plus `TestDetectContainer_DockerenvSymlinkCounts`
  in `internal/cli/doctor/env_internal_test.go`; three serial
  integration tests in `internal/cli/integration/doctor_cmd_test.go`
  (`TestDoctorReport_EnvLinePresent_DevcontainerCase`,
  `TestDoctorReport_EnvLine_RespectsFalsyEnvVar`,
  `TestDoctorReport_EnvLine_InformationalOnly`). Red phase visible
  in dev-loop output before commit (`undefined: detectContainer` +
  missing `env:` line). Phase walked `red → green → refactor →
  done`; status promoted `open → met`.

### AC-2 — Shadow-mount status check

- Red→green bundled (`a8063474`): extended `internal/cli/doctor/env.go`
  with the `mountState` enum (ok / empty / missing / error /
  unknown), `shadowMountCountCap = 100`, `shadowMountStatus(home)`
  probe, `pluginsTargetPath()` helper, and `renderMountLine()`
  renderer. `DoctorReport` gates the `plugin-index-mount:` line
  on `InContainer()` from AC-1; uses `os.UserHomeDir()` and falls
  through to a `plugin-index-mount: <err>` line on resolution
  failure. Test set: 6 sub-tests in `TestShadowMountStatus` (ok,
  empty, missing, regular-file-not-dir, hidden-entries-excluded,
  100+ cap) plus 6 sub-tests in `TestRenderMountLine` pinning
  render shape across all states; two serial integration tests
  in `doctor_cmd_test.go` (`TestDoctorReport_ShadowMount_PluginIndexLineGatedOnContainer`
  and `TestDoctorReport_ShadowMount_ReportsMissingAndOK`).
  Phase walked `red → green → refactor → done`; status promoted
  `open → met`.

## Decisions made during implementation

- **`detectContainer` takes inputs explicitly, not via package
  globals.** The unexported `detectContainer(dockerenvPath,
  devcontainerEnv)` shape was chosen over a package-level
  `var dockerenvPath = "/.dockerenv"` (CLAUDE.md forbids
  package-level mutable state and "production patterns
  introduced purely to satisfy test-injection"). The cost is a
  trivial pass-through `InContainer()` zero-arg wrapper; the
  benefit is the unit test exercises the full signal-combination
  matrix without touching the FS root.
- **Integration-test coverage scoped to wiring; combinatorial
  coverage lives in unit tests.** The spec wrote the integration
  test as a 4-combo table-driven set "via t.Setenv". In practice
  `t.Setenv` only controls AIWF_DEVCONTAINER — the `/.dockerenv`
  side is fixed at the FS root and uncontrollable from the
  integration boundary. The full combinatorial coverage moved to
  `TestDetectContainer` in the doctor package (t.TempDir-rooted
  dockerenv fixture); the integration tests confirm DoctorReport
  wiring only (env: line present, env-var respected, never
  increments problems). Spec's "code references" line updated to
  note the split.
- **Removed redundant `TestDoctorReport_ShadowMount_InformationalOnly`
  integration test.** The "never increments problems" assertion
  tripped on a pre-existing actor-resolution path (`HOME`
  redirection broke `git config user.email` lookup) unrelated to
  the AC-2 code path. The assertion was structurally tautological
  (no `problems++` exists in the AC-2 branch); the same contract
  is verifiable by inspection and by `TestDoctorReport_EnvLine_InformationalOnly`
  for the AC-1 line. Removed rather than worked around with
  repo-local git config seeding.
- **Red→green bundled into one commit per AC.** Same precedent as
  M-0134/AC-1: the unit tests reference unexported types
  (`detectContainer`, `shadowMountStatus`, `mountState*`
  constants), so a discrete red commit would leave the doctor
  package un-buildable and break every downstream package
  importing it. Bundling preserves the kernel's "no `--no-verify`
  unless explicitly requested" rule while keeping the red→green
  progression visible in the dev loop (red output captured before
  each commit). The phase-promote commits (`--phase green/refactor/done`)
  do record the phase walk explicitly in `aiwf history`.

## Validation

- `make test`: full suite green (exit 0, no failures across all
  `internal/...` and `cmd/...` packages).
- `aiwf check`: 0 errors; 23 warnings, all pre-existing or
  natural consequences of state (the new `epic-active-no-drafted-milestones`
  warning is the expected state — M-0135 was the last drafted
  milestone in E-0035; resolves at epic wrap).
- `golangci-lint run ./internal/cli/doctor/ ./internal/cli/integration/`:
  0 issues (one `gocritic:unnamedResult` finding caught mid-flight
  on `InContainer()` and `detectContainer()` return tuples,
  resolved by naming the results `inContainer bool, label string`).
- **Operator smoke verification in this very worktree.**
  Built `/tmp/aiwf-m0135` from the milestone branch and ran
  `aiwf doctor` against `/workspaces/aiwf-devcontainer-dev-loop`.
  Output's first three lines:
  ```
  binary:    (devel) (working-tree build)
  env:       devcontainer (/.dockerenv + AIWF_DEVCONTAINER)
  plugin-index-mount: ok (5 plugin entries cached)
  ```
  Both new lines emit at the documented position; `env:` reports
  both signals as expected; `plugin-index-mount: ok` counts the
  5 plugin subdirs in `/home/vscode/.claude/plugins/`; problems
  unchanged from the pre-M-0135 baseline.

## Reviewer notes

- **Branch coverage on `env.go`**: every reachable conditional
  branch in the new code is exercised by `TestDetectContainer`
  (7 cases), `TestShadowMountStatus` (6 cases), and
  `TestRenderMountLine` (6 cases). Two defensive paths in
  `shadowMountStatus` (non-NotExist `os.Stat` error, mid-iter
  `os.ReadDir` error) are unreachable from test fixtures — they
  propagate through the tested `renderMountLine(mountStateError, …)`
  path, so the rendered output is covered even though the error
  origin isn't synthesised. Acceptable per the "test reasonable
  branches; defensive paths verified at the next observable
  boundary" convention.
- **The host-case `plugin-index-mount:` omission cannot be
  integration-tested from inside the devcontainer.** `/.dockerenv`
  exists at the FS root and can't be stripped without root
  privilege; `t.Setenv` only controls one of the two signals. The
  omission contract is verifiable by code inspection (the
  `if inContainer { … }` gate around the line is a single
  reachable site) and by the unit-level coverage of `InContainer()`
  returning `(false, "host")` from `TestDetectContainer`. If we
  later move detection into something path-injectable for
  DoctorReport (constructor injection via DoctorOptions), a
  matching `host`-case integration test becomes trivial. Not done
  this milestone — would be a production pattern introduced
  purely to satisfy test injection, which CLAUDE.md flags.
- **No CHANGELOG entry needed** — this is a doctor-output addition
  that surfaces existing state, not a verb or behavior change a
  consumer would need to know about pre-upgrade. Consumers see
  the two new informational lines on their next `aiwf doctor`
  run, no migration or workflow change.
- **Hybrid worktree model held up cleanly across this milestone**:
  branch off main → bundled red+green commits + phase-promote
  commits → operator smoke from the worktree → wrap via `--no-ff`
  back to main with the worktree's epic branch fast-forwarded
  afterward. Same shape as M-0133 and M-0134; no friction.

## Deferrals

- None this milestone. The two out-of-scope items listed in the
  Approach section (container-aware advice strings;
  deeper plugin-index introspection) remain candidates for a
  future gap if friction surfaces in practice; no file opened
  today because no friction has been observed since M-0132 / M-0133
  closed.
