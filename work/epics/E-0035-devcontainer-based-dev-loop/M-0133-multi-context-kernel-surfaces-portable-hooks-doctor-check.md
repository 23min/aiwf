---
id: M-0133
title: 'Multi-context kernel surfaces: portable hooks + doctor check'
status: in_progress
parent: E-0035
tdd: required
acs:
    - id: AC-1
      title: Portable hook binary lookup via PATH at hook execution time
      status: met
      tdd_phase: done
    - id: AC-2
      title: aiwf update from a worktree writes to shared hooks directory
      status: met
      tdd_phase: done
    - id: AC-3
      title: aiwf doctor recommended-plugins reads enabledPlugins not installed-plugins
      status: met
      tdd_phase: done
---
## Goal

Eliminate the three kernel surfaces that assume single-context dev
and bite every multi-context session (host ↔ devcontainer ↔
worktree). The "multi-context tax" — hook-path tug-of-war, inert
worktree hook writes, and false-positive plugin warnings — currently
forces operators to remember per-context discipline and re-run
`aiwf init` / `aiwf update` when switching environments, or
explicitly ignore warnings that are known-false. Land three surgical
fixes so the same checkout works frictionlessly across all contexts
after one `aiwf init`.

## Approach

Three small, surgical kernel-side changes, each closing one gap,
shipped as one milestone so the multi-context dev story lands as
one coherent improvement.

1. **Portable hook binary lookup** (closes [G-0135](../../../gaps/G-0135-hook-path-hardcoded-at-install-time-breaks-across-gopath-environments.md)).
   Replace the install-time absolute `aiwf` path baked into
   `pre-commit` / `pre-push` / `post-commit` hooks with a PATH-relative
   `command -v aiwf` resolution at hook execution time. Same hook
   contents work from any environment whose PATH contains `aiwf`.
   Touches the hook-installation templates in `internal/initrepo/initrepo.go`
   (`preHookScript`, `preCommitHookScript`, `postCommitHookScript`).
   Matches the design-decisions §"Contracts" posture that the
   engine doesn't ship binaries — PATH lookup is the consumer's job.

2. **`aiwf update` writes to shared hooks dir from worktrees**
   (closes [G-0136](../../../gaps/G-0136-aiwf-update-from-a-worktree-writes-hooks-git-ignores.md)).
   When `aiwf update` runs in a worktree, resolve the hooks directory
   via `git rev-parse --git-common-dir` so the write lands at the
   main repo's shared `.git/hooks/`, not the inert per-worktree
   `.git/worktrees/<id>/hooks/`. Per-worktree hook divergence is
   not a current use case; the shared-write model matches the actual
   semantic (aiwf hooks are repo-level policy). Operator output
   explicitly states the write affects all worktrees.

3. **`aiwf doctor` recommended-plugins reads `enabledPlugins`**
   (closes [G-0138](../../../gaps/G-0138-aiwf-doctor-recommended-plugin-check-false-positives-across-multi-context-dev.md)).
   Switch the recommended-plugins check's source of truth from
   machine-local `~/.claude/plugins/installed_plugins.json`
   (path-strict; false-positives across worktrees, containers, and
   re-clones) to `<rootDir>/.claude/settings.json`'s `enabledPlugins`
   map (in the source tree; path-independent by construction). Drops
   the path-equality comparison entirely. Secondary: update the
   install-advice string to steer the operator toward PROJECT scope
   (the bare `claude /plugin install <name>@<marketplace>` CLI form
   defaults to user scope per Claude Code docs; only the interactive
   `/plugin` menu offers a project-scope choice).

Each change carries a mechanical assertion in the package that owns
the affected code. The three changes touch independent code paths
(hook template / install verb / doctor verb), so implementation order
is flexible.

## Acceptance criteria

ACs land via `aiwf add ac M-NNNN`; the three are summarised above.

### AC-1 — Portable hook binary lookup via PATH at hook execution time

**Pass criterion**: the hook templates emitted by `aiwf init` and
`aiwf update` (currently `pre-commit`, `pre-push`, `post-commit`)
contain no absolute path literal to the `aiwf` binary. Instead the
rendered hook resolves `aiwf` via a PATH-relative `command -v aiwf`
lookup at hook execution time. `pre-push` and `pre-commit` exit
non-zero with a clear stderr message when aiwf is not on PATH
(validation hooks must not silently skip); `post-commit` logs the
not-found message to stderr but exits 0 (STATUS.md regen is best-
effort and must not disturb a successful commit).

**Edge cases**: PATH unset entirely (the validation hooks exit
non-zero with a clear message, not silent skip); `aiwf` absent from
PATH (same); a stale absolute path inherited from a pre-fix install
(the template renderer rewrites it on the next `aiwf update` and
`aiwf doctor` surfaces the pre-fix shape as `run aiwf update to
switch to PATH lookup`).

**Code references**: hook template generators in
`internal/initrepo/initrepo.go` (`preHookScript`, `preCommitHookScript`,
`postCommitHookScript`); the now-unused `resolveExecutable` helper
removed. Doctor's hook-validity surface in
`internal/cli/doctor/doctor.go` (`appendHookReport`,
`appendPreCommitHookReport`, `appendPostCommitHookReport`) updated
to recognize both the new (`command -v aiwf`) and pre-G-0135
(baked-path) shapes via `exec.LookPath`. Regression tests:
`internal/initrepo/multi_context_test.go` pins the PATH-lookup shape
across all three hook templates; `internal/initrepo/testdata/pre-push.golden`
pins byte-exact contents; brownfield + chain tests in
`internal/initrepo/brownfield_test.go`, `precommit_test.go`,
`postcommit_test.go` updated to place the shim on PATH (since
`execPath` is no longer threaded through); doctor branch-coverage
tests in `internal/cli/integration/doctor_cmd_test.go`
(`TestDoctorReport_HookOK_AiwfNotOnPATH`,
`TestDoctorReport_PreG0135ShapeStillValid`).

### AC-2 — aiwf update from a worktree writes to shared hooks directory

**Pass criterion**: when `aiwf update` runs with cwd inside a git
worktree, the hook write resolves the target directory via
`git rev-parse --path-format=absolute --git-common-dir` and writes
to `<common-dir>/hooks/`, not the inert
`<common-dir>/worktrees/<id>/hooks/`. Operator output explicitly
states that the write affects all worktrees of the repo.

**Edge cases**: invoked from the main checkout (no worktree active) —
behaves identically to before (gitDir == commonDir, no regression);
invoked from a workdir that isn't a git repo at all — error surfaces
via `HooksDir` / `InWorktree`, not silently falling back to a wrong
path; pre-existing per-worktree hooks left behind by an older
`aiwf update` are not touched (this AC fixes the writer, not the
cleanup of legacy inert hooks — file a follow-up gap if cleanup
proves necessary). When `core.hooksPath` is set, `HooksDir` still
honors it — operators who want per-worktree divergence opt in via
that knob.

**Code references**: `internal/gitops/gitops.go` — `HooksDir` falls
back to `commonGitDir/hooks` (new helper using `--git-common-dir`)
instead of the per-worktree `gitDir/hooks`; new `InWorktree` helper
compares them. `internal/cli/update/update.go` — `Run` prints the
affects-all-worktrees notice via `gitops.InWorktree`. Regression
tests: `internal/gitops/gitops_test.go::TestHooksDir/worktree_falls_back_to_common-dir_hooks_(G-0136)`
(unit) creates a worktree and asserts `HooksDir` returns the shared
path; `internal/cli/integration/update_cmd_test.go::TestRun_UpdateFromWorktree_WritesSharedHooks`
(integration) creates a worktree via `git worktree add`, removes
the shared `.git/hooks/pre-push`, runs `aiwf update` with cwd in
the worktree via `cli.Execute`, and asserts (a) shared
`.git/hooks/pre-push` was reinstalled, (b) per-worktree
`.git/worktrees/wt/hooks/pre-push` was not created, (c) captured
stdout contains the "affects all worktrees" notice.

### AC-3 — aiwf doctor recommended-plugins reads enabledPlugins not installed-plugins

**Pass criterion**: `aiwf doctor`'s recommended-plugins check sources
truth from `<rootDir>/.claude/settings.json`'s `enabledPlugins` map;
the path-strict `~/.claude/plugins/installed_plugins.json` comparison
is removed entirely. A plugin listed as `enabledPlugins: { "name@market": true }`
is reported as satisfied regardless of `installed_plugins.json`
contents or rootDir path. Secondary: when the warning fires (plugin
absent / disabled), the install-advice string steers the operator
toward PROJECT scope via the interactive `/plugin` menu (the bare
CLI form defaults to user scope per Claude Code docs).

**Edge cases**: missing `.claude/settings.json` (treat as no plugins
enabled and fire the warning with full advice); malformed JSON
(surface as a clear `plugins:` line naming the file, do not silently
claim plugin status either way); `enabledPlugins` value of `false`
or absent key (treat as disabled).

**Code references**: `internal/cli/doctor/doctor.go` —
`appendRecommendedPluginsReport` (replaces the prior
`pluginstate.Load(home) + HasProjectScope(plugin, rootDir)` pair
with the new `loadEnabledPlugins(rootDir)` helper that JSON-reads
`<rootDir>/.claude/settings.json`); the `pluginstate` import dropped
from doctor (the package is still alive in case other consumers need
it). `internal/cli/doctor/selfcheck.go` — step 7's silencing fixture
now writes `.claude/settings.json` (`writeEnabledPluginsForSelfCheck`)
instead of `installed_plugins.json` (`writeInstalledPluginsForSelfCheck`,
removed). Regression tests in
`internal/cli/integration/doctor_cmd_test.go`:
`TestDoctorReport_RecommendedPlugins_EnabledInSettings_NoWarning`
pins the new source-of-truth happy path,
`TestDoctorReport_RecommendedPlugins_AdviceMentionsProjectScope`
pins the PROJECT-scope advice,
`TestDoctorReport_RecommendedPlugins_MalformedSettings_Error`
pins error-on-bad-json. Obsolete tests removed:
`AllInstalledForProject_NoWarning`, `InstalledElsewhereStillWarns`,
`CorruptedIndex_EmitsAdvisory`, and the
`writeInstalledPluginsFixture` helper (installed_plugins.json
path-equality semantics no longer apply).

## Work log

### AC-1 — Portable hook binary lookup (G-0135)

- Red (`a916fec9`): `TestHookScripts_UsePATHResolution` pins `command -v aiwf` shape + fail-loud + sentinel-absent across all three hook templates. Three subtests fail against current code.
- Green (`facf6533`): rewrite `preHookScript` / `preCommitHookScript` / `postCommitHookScript` to use the new prelude; update `pre-push.golden`; update brownfield + chain tests to place shim on PATH; teach doctor's three hook-report functions to recognize both shapes.
- Refactor (`d5bdb2ba`): drop the vestigial `execPath` parameter from all three template signatures; remove `resolveExecutable`; collapse all callers.
- Branch-coverage audit (`ce127abf`): two new doctor tests in `internal/cli/integration/doctor_cmd_test.go` covering the post-G-0135 "not found on PATH" path and the pre-G-0135 "still valid" path. Hook-report coverage rose from ~55-61% to 71-78%.

### AC-2 — `aiwf update` writes to shared hooks directory (G-0136)

- Red (`9ec4bcfe`): `TestHooksDir/worktree…` (gitops unit) + `TestRun_UpdateFromWorktree_WritesSharedHooks` (cli integration). Both fail today — hooks land in the inert per-worktree dir; no operator notice.
- Green (`436a1a8d`): add `commonGitDir` + `InWorktree` helpers in gitops; rewrite `HooksDir`'s fallback to use commonGitDir; teach `update.Run` to print the affects-all-worktrees notice via `gitops.InWorktree`.
- Branch-coverage audit (`e551293e`): add a non-git-workdir subtest to `TestHooksDir` covering `commonGitDir` and `InWorktree` error paths.

### AC-3 — Doctor reads `enabledPlugins` from `.claude/settings.json` (G-0138)

- Red (`ad55e63c`): three new tests pin the new contract (`EnabledInSettings_NoWarning`, `AdviceMentionsProjectScope`, `MalformedSettings_Error`).
- Green (`481d4f4e`): rewrite `appendRecommendedPluginsReport` against the new `loadEnabledPlugins` helper; drop the `pluginstate` import; update selfcheck step 7's silencing fixture to use `writeEnabledPluginsForSelfCheck`; remove obsolete tests (`AllInstalledForProject_NoWarning`, `InstalledElsewhereStillWarns`, `CorruptedIndex_EmitsAdvisory`) and the `writeInstalledPluginsFixture` helper. Selfcheck step 1 verifyOutput updated to assert the new PROJECT-scope advice text.

## Decisions made during implementation

- **Post-commit hook stays tolerant on missing aiwf.** AC-1's fail-loud rule applies to pre-push and pre-commit (validation must not silently skip). post-commit's STATUS.md regen is best-effort and should not disturb a successful commit, so it logs the not-found message to stderr but exits 0.
- **`execPath` parameter dropped, not retained.** Once `command -v aiwf` runs at hook-fire time, the install-time path has no place in the template. Keeping the parameter "ignored" would have left a backwards-compat shim with no use case; per the kernel's "no half-finished implementations" rule, drop it cleanly.
- **Doctor recognizes both hook shapes.** A consumer with an older install still gets an actionable diagnostic ("pre-G-0135 shape, run `aiwf update` to switch to PATH lookup") rather than a confusing "malformed" error. The two paths cohabit in doctor's hook-report functions; the next major version can prune the pre-G-0135 path.
- **Install advice steers to the interactive `/plugin` menu, not `--scope project`.** The original gap text proposed `--scope project`, but per CLAUDE.md the bare CLI form has no project-scope flag — only the interactive menu offers the choice. Advice updated to reflect that constraint.

## Validation

- `make ci` (the project's operational gate): **30/30 selfcheck steps pass**, including the renamed AC-3 step `doctor recommended-plugins fixture: warning silent after enable in settings.json`. Build green; `golangci-lint` clean; `go vet` clean; full test suite green.
- `aiwf check`: 0 errors, 23 warnings (all pre-existing — no new findings introduced by this milestone).
- **Operator smoke in this very worktree** (`/workspaces/aiwf-devcontainer-dev-loop`, a linked worktree of the main repo):
  - `aiwf doctor` (pre-update): all three hooks reported as `pre-G-0135 shape, run aiwf update to switch to PATH lookup`; no `recommended-plugin-not-installed` warning (AC-3 working — `.claude/settings.json` `enabledPlugins` is the source of truth and matches the recommended set).
  - `aiwf update` (from the worktree): hook writes land at `../aiwf/.git/hooks/{pre-push,pre-commit,post-commit}` (the shared dir, not the per-worktree path), with `Detail` strings reading `exec` `command -v aiwf` `... (PATH-relative)`. Output ends with the affects-all-worktrees notice: *"running from a linked worktree. Hook writes go to the shared `.git/hooks/` directory; this update affects all worktrees of the repo."*
  - `aiwf doctor` (post-update): all three hooks reported as `ok (resolves to /go/bin/aiwf)` — the post-G-0135 shape recognized.
- **`docs/pocv3/design/design-decisions.md`** updated: the `doctor` config block now documents the `<rootDir>/.claude/settings.json` source of truth (replacing the stale `~/.claude/plugins/installed_plugins.json` description).

## Reviewer notes

- **Doctor still carries the pre-G-0135 detection path** (in `appendHookReport`, `appendPreCommitHookReport`, `appendPostCommitHookReport`). This is deliberate: consumers with an older `aiwf init` still get an actionable `run aiwf update to switch to PATH lookup` advisory instead of a confusing `malformed` error. The next major version can prune that arm. Tests cover both shapes (`TestDoctorReport_PreG0135ShapeStillValid`).
- **The `pluginstate` package is no longer imported by `internal/cli/doctor/doctor.go`** but the package and its tests remain alive. It's a small read helper and a future caller may emerge; deletion is deferred (mirrored in Deferrals below).
- **AC body forward-reference fix.** The original AC bodies (authored at allocation time) pointed at a single test file `internal/policies/multi_context_tax_test.go`. The tests actually distributed across `internal/initrepo/multi_context_test.go`, `internal/gitops/gitops_test.go`, and `internal/cli/integration/{doctor_cmd,update_cmd}_test.go` based on the code surface each AC touches. The body was rewritten via `aiwf edit-body M-0133` to reflect the real test locations.
- **Branch-coverage stance.** AC-1's new doctor branches (`exec.LookPath` fail + pre-G-0135 shape) and AC-2's new gitops helpers (`commonGitDir`, `InWorktree`) added explicit tests covering the load-bearing paths. The remaining uncovered statements are pure error-wrap pass-throughs (`if err != nil { return err }` shapes) matching the existing pattern in `GitDir`/`HooksDir` — not test theatre, but a deliberate match to the package's tolerance for trivial error-return coverage gaps.
- **No change to the FSM, no schema bumps, no kernel-surface additions.** The three changes are surgical: a template rewrite, a path resolution swap, and a JSON-source swap. Backwards-compatible for consumers with the new binary.

## Deferrals

- **`aiwf init` from a worktree** also installs hooks; the affects-all-worktrees notice currently fires only from `aiwf update`. The same scope question applies but lands quietly. File a follow-up gap if real friction appears.
- **`pluginstate` package** is now unused by doctor. The package is small and may find another use; deletion is deferred until it's confirmed dead across the kernel.
