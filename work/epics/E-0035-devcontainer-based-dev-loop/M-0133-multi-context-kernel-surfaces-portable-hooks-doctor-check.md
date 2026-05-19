---
id: M-0133
title: 'Multi-context kernel surfaces: portable hooks + doctor check'
status: in_progress
parent: E-0035
tdd: required
acs:
    - id: AC-1
      title: Portable hook binary lookup via PATH at hook execution time
      status: open
      tdd_phase: green
    - id: AC-2
      title: aiwf update from a worktree writes to shared hooks directory
      status: open
      tdd_phase: red
    - id: AC-3
      title: aiwf doctor recommended-plugins reads enabledPlugins not installed-plugins
      status: open
      tdd_phase: red
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
   Touches the hook-installation template (in `internal/cli/install/`
   or wherever the hook generator lives; verified on first read).
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
   the path-equality comparison entirely. Secondary: fix the
   install-advice string to include `--scope project` (matches the
   CLAUDE.md operator-setup recipe; the bare CLI form defaults to
   user-scope per Claude Code docs).

Each change carries its own AC with a mechanical assertion under
`internal/policies/`. The three changes touch independent code paths
(hook template / install verb / doctor verb), so implementation order
is flexible.

## Acceptance criteria

ACs land via `aiwf add ac M-NNNN`; the three are summarised above.

### AC-1 — Portable hook binary lookup via PATH at hook execution time

**Pass criterion**: the hook templates emitted by `aiwf init` and
`aiwf update` (currently `pre-commit`, `pre-push`, `post-commit`)
contain no absolute path literal to the `aiwf` binary. Instead the
rendered hook resolves `aiwf` via a PATH-relative `command -v aiwf`
lookup at hook execution time, with a not-found fallback that exits
non-zero with a clear message naming the missing binary.

**Edge cases**: PATH unset entirely (must exit non-zero with a clear
message, not silently no-op); `aiwf` absent from PATH (must exit
loud, not skip the check); a stale absolute path inherited from a
pre-fix install (the template renderer must rewrite it on the next
`aiwf update`, not preserve it).

**Code references**: hook template generator under
`internal/cli/install/` (exact file confirmed on first read);
regression test in `internal/policies/multi_context_tax_test.go`
asserting the rendered hook text contains no absolute-path
`aiwf`-binary literal and contains the `command -v aiwf` shape.

### AC-2 — aiwf update from a worktree writes to shared hooks directory

**Pass criterion**: when `aiwf update` runs with cwd inside a git
worktree, the hook write resolves the target directory via
`git rev-parse --git-common-dir` and writes to `<common-dir>/hooks/`,
not the inert `<common-dir>/worktrees/<id>/hooks/`. Operator output
explicitly states that the write affects all worktrees of the repo.

**Edge cases**: invoked from the main checkout (no worktree active) —
must behave identically to today (no regression on the existing
happy path); invoked when `git rev-parse --git-common-dir` resolution
fails (corrupt or non-repo cwd) — must error cleanly with a useful
message rather than silently fall back to a wrong path; pre-existing
per-worktree hooks left behind by an older `aiwf update` should not
be touched (this AC fixes the writer, not the cleanup of legacy
inert hooks — file a follow-up gap if cleanup proves necessary).

**Code references**: install/update logic in `internal/cli/install/`
(the hook-materialization helper); regression test in
`internal/policies/multi_context_tax_test.go` creates a worktree via
`git worktree add`, runs `aiwf update` with cwd in the worktree, and
asserts (a) shared `.git/hooks/<name>` was touched, (b) per-worktree
`.git/worktrees/<id>/hooks/<name>` was not created, (c) stdout/stderr
contains the "affects all worktrees" notice.

### AC-3 — aiwf doctor recommended-plugins reads enabledPlugins not installed-plugins

**Pass criterion**: `aiwf doctor`'s recommended-plugins check sources
truth from `<rootDir>/.claude/settings.json`'s `enabledPlugins` map;
the path-strict `~/.claude/plugins/installed_plugins.json` comparison
is removed entirely. A plugin listed as `enabledPlugins: { "name@market": true }`
is reported as satisfied regardless of `installed_plugins.json`
contents or rootDir path. Secondary: when the warning fires (plugin
absent / disabled), the install-advice string contains
`--scope project` to match the operator-setup recipe in CLAUDE.md
(the bare `claude /plugin install <p>@<m>` form defaults to user
scope per Claude Code docs).

**Edge cases**: missing `.claude/settings.json` (treat as no plugins
enabled and fire the warning with full advice); malformed JSON
(return a clear error naming the file, do not silently claim plugin
status either way); `enabledPlugins` schema variations (boolean
`true` vs object form — accept both, treat `false` or absent key as
disabled).

**Code references**: `internal/cli/doctor/doctor.go` —
`appendRecommendedPluginsReport` (replace
`pluginstate.Load(home) + HasProjectScope(plugin, rootDir)` with a
JSON read of `<rootDir>/.claude/settings.json`); regression test in
`internal/policies/multi_context_tax_test.go` exercises four fixture
trees (enabled / disabled / missing-settings / malformed-settings)
and asserts the doctor's report shape for each.

