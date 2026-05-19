---
id: M-0132
title: Land .devcontainer skeleton (features-first, Go base, project-scope plugins)
status: draft
parent: E-0035
tdd: none
acs:
    - id: AC-1
      title: devcontainer.json declares base image, features, mounts, workspace mount
      status: met
    - id: AC-2
      title: devcontainer-lock.json pins feature SHAs
      status: met
    - id: AC-3
      title: initialize.sh creates host-side symlinks and cites claude-code#31388
      status: met
    - id: AC-4
      title: init.sh runs idempotently with the agreed install + banner blocks
      status: open
    - id: AC-5
      title: .devcontainer/README.md ships operator-facing usage with named sections
      status: open
    - id: AC-6
      title: CLAUDE.md Operator setup gains Devcontainer subsection with shadow-mount note
      status: open
    - id: AC-7
      title: devcontainer-build-smoke.sh exists with build + Go-version check
      status: cancelled
    - id: AC-8
      title: devcontainer-ci-smoke.sh exists with make-ci-inside-container check
      status: cancelled
---
## Goal

Land a working `.devcontainer/` skeleton that becomes the canonical
dev surface for aiwf. The container builds clean from cold cache,
`make ci` is green inside it, plugin install completes at PROJECT
scope without corrupting the host's macOS-pathed plugin index, and
`git push` works from inside via the host-bound gh credential
helper. The macOS host-fallback path remains intact and untouched
— this milestone is additive.

## Approach

Features-first composition on Microsoft's first-party Go base image,
host-side symlink dance per the FlowTime precedent, postcreate
banner for the one manual plugin-install step per the
claude-code#31388 workaround. Concretely:

- **Base image:** `mcr.microsoft.com/devcontainers/go:2-1.25-bookworm`.
  First-party Go image avoids the FlowTime pain of installing Go
  on a stranger base image.
- **Features:** `common-utils:2` (zsh + oh-my-zsh + vscode user),
  `github-cli:1`, `node:1` (v22 for Playwright). SHAs pinned in
  `devcontainer-lock.json`.
- **Workspace mount:** `${localWorkspaceFolder}/..` → `/workspaces`.
  One level up so the rituals plugin source
  (`~/Projects/ai-workflow-rituals/`) is reachable for cross-repo
  testing per CLAUDE.md.
- **Mount mounts (the /tmp symlink dance):**
    - `~/.claude` → `/home/vscode/.claude` (full state shared
      with host).
    - `~/.claude-linux/plugins` → `/home/vscode/.claude/plugins`
      (Linux-specific plugin index; shadows the host's
      macOS-pathed index per claude-code#31388).
    - `~/.config/gh` → `/home/vscode/.config/gh` (gh auth shared).
- **`initializeCommand` (host-side):** creates `/tmp/.claude-mount`,
  `/tmp/.claude-plugins-mount`, `/tmp/.gh-mount` symlinks so
  `devcontainer.json`'s `mounts:` entries reference stable `/tmp`
  paths (devcontainer.json can't expand `$HOME` in mount sources).
- **`postCreateCommand` (container-side, idempotent):**
    1. `git config --global user.name "Peter Bruinsma"` /
       `user.email "peter@23min.com"` — match host identity so
       aiwf trailers stay consistent.
    2. Fix gh credential helper path per Liminara's precedent:
       `git config --global credential.https://github.com.helper "!gh auth git-credential"`.
    3. Install golangci-lint v2.11.4 (matches CI's pinned
       version), gofumpt, govulncheck.
    4. Install Claude Code CLI if not present
       (`curl -fsSL https://claude.ai/install.sh | bash`).
    5. `make install-hooks` (kernel's pre-commit chain).
    6. `go install ./cmd/aiwf` + `aiwf init` (idempotent;
       materializes hooks and gitignored skill adapters).
    7. Env-gated Playwright install: if
       `${AIWF_DEVCONTAINER_E2E:-false}` is `true`, run
       `(cd e2e/playwright && npm install && npx playwright install chromium)`.
       Default off — most contributors aren't touching the HTML
       renderer.
    8. Print a banner with two instructions: (a)
       `/plugin marketplace add 23min/ai-workflow-rituals`, then
       install both `aiwf-extensions` and `wf-rituals` at
       PROJECT scope via the interactive `/plugin` menu (CLI
       form defaults to USER scope — wrong); (b) the
       `AIWF_DEVCONTAINER_E2E` env var for opting into
       Playwright next rebuild.
- **VS Code config:** extensions = `golang.go`,
  `redhat.vscode-yaml`, `anthropic.claude-code`,
  `mhutchie.git-graph`, `editorconfig.editorconfig`,
  `github.vscode-github-actions`,
  `shd101wyy.markdown-preview-enhanced`. Settings: Go
  formatOnSave; yaml schema mapping for `aiwf.yaml` if a schema
  lands later.
- **No commit signing** — host doesn't sign; container parity
  preserved.
- **Auth model:** HTTPS + gh credential helper. No SSH mount.
- **Trust posture:** workspace mount is read-write (the
  `${localWorkspaceFolder}/..` mount widens to all of
  `~/Projects/`); same trust model as a host shell.
- **macOS host-fallback path stays intact.**
  `scripts/sign-and-run.sh`, the in-test `codesign` blocks, the
  `-parallel 8` cap all remain; the CLAUDE.md DO/DON'T section
  is reframed in a later milestone.
- **Operator path is VS Code "Reopen in Container".** The
  Dev Containers extension drives image build + container start
  + `init.sh` execution; no standalone `@devcontainers/cli`
  install needed on the host for day-to-day use. The smoke
  scripts originally planned for AC-7/AC-8 belong with the
  future CI matrix (sibling milestone under E-0035, which does
  need the standalone CLI).

## Acceptance criteria

Six ACs cover the structural shape of the .devcontainer/ files
and the CLAUDE.md operator-setup subsection. All six are pure
structural assertions mechanized via `internal/policies/`. ACs
scaffold below as `### AC-N — <title>` sub-elements; each
carries its own pass criterion, edge cases, and code references.

AC-7 and AC-8 (smoke scripts for `devcontainer build` and
in-container `make ci`) were cancelled mid-implementation when
the design conversation surfaced that the canonical operator
path is VS Code's "Reopen in Container" + integrated-terminal
`make ci` — no standalone `@devcontainers/cli` install needed
on the host. The smoke-script work moves to the sibling "CI
matrix integration (Docker-in-Docker)" milestone under E-0035,
where the standalone CLI is genuinely required by a CI runner
that has no VS Code.

### AC-1 — devcontainer.json declares base image, features, mounts, workspace mount

The verb-time projection of `.devcontainer/devcontainer.json`
JSON-parses cleanly and contains the agreed structural shape.
**Pass criterion**: file parses as JSON; `image` is
`mcr.microsoft.com/devcontainers/go:2-1.25-bookworm`; `features`
contains entries for
`ghcr.io/devcontainers/features/common-utils:2`,
`ghcr.io/devcontainers/features/github-cli:1`, and
`ghcr.io/devcontainers/features/node:1`; `workspaceMount.source`
expands to `${localWorkspaceFolder}/..` (siblings visible);
`mounts` contains entries for `/tmp/.claude-mount`,
`/tmp/.claude-plugins-mount`, and `/tmp/.gh-mount` targeting the
three named in-container paths; `remoteUser` is `vscode`.
**Edge cases**: extra entries fine; missing any of the three named
features fails; image string mismatch fails; mount target paths
inverted (host vs. container side) fails; `workspaceMount.source`
expanding only to `${localWorkspaceFolder}` (no `..`) fails because
sibling repos would be invisible.
**Code references**: assertion at
`internal/policies/devcontainer_shape_test.go` (new); reads the
file via `os.ReadFile`, parses via `encoding/json`, asserts each
named field via structured navigation (not flat substring match).

### AC-2 — devcontainer-lock.json pins feature SHAs

`.devcontainer/devcontainer-lock.json` exists and pins resolved
SHAs for every feature declared in `devcontainer.json`, matching
FlowTime's reproducibility precedent.
**Pass criterion**: file parses as JSON; the `features` object
contains a key for each feature declared in `devcontainer.json`;
each entry has non-empty `version`, `resolved` (with `@sha256:...`
suffix), and `integrity` fields.
**Edge cases**: missing feature in lockfile fails; lockfile entry
without `resolved` SHA fails; mismatch between feature ids in
`devcontainer.json` and `devcontainer-lock.json` (e.g., feature
added in `.json` but not regenerated into `-lock.json`) fails.
**Code references**: assertion at
`internal/policies/devcontainer_lock_test.go` (new);
cross-validates the two files by walking `devcontainer.json`'s
`features` keys and asserting each is present and pinned in the
lockfile.

### AC-3 — initialize.sh creates host-side symlinks and cites claude-code#31388

`.devcontainer/initialize.sh` is the host-side `initializeCommand`
hook. It must create the stable `/tmp` symlinks the devcontainer
mounts reference, and document the shadow-mount workaround inline.
**Pass criterion**: file exists with mode `0755`; bash header
(`#!/usr/bin/env bash` + `set -euo pipefail`); contains three
`ln -sfn "$HOME/..." /tmp/<name>` lines for `.claude-mount`,
`.claude-plugins-mount`, and `.gh-mount`; contains a comment block
above the symlink section that names
`anthropics/claude-code#31388` by URL, names the macOS-vs-Linux
plugin-index path mismatch as the cause, and instructs the reader
to remove the shadow-mount once #31388 ships a fix.
**Edge cases**: missing any of the three symlinks fails; comment
block in wrong location (e.g., at file end, after the symlinks)
fails per the structural-assertion rule; missing the upstream URL
literal fails; comment present but not naming "shadow-mount" or
"plugin index" as the concept fails the structural intent check.
**Code references**: assertion at
`internal/policies/devcontainer_initialize_script_test.go` (new);
reads the file, asserts mode via `os.Stat`, asserts the three
`ln -sfn` lines via structured regex anchored to the symlink
section, asserts the comment block precedes them and contains
the upstream URL.

### AC-4 — init.sh runs idempotently with the agreed install + banner blocks

`.devcontainer/init.sh` is the in-container `postCreateCommand`
hook. It performs the install and configuration sequence the
milestone Approach section enumerates, idempotently (safe to
re-run on `updateContentCommand` rebuilds).
**Pass criterion**: file exists with mode `0755`; contains
named-section comments for each of: git config (user.name +
user.email matching the host identity), gh credential helper
(`!gh auth git-credential` rewrite per Liminara's precedent),
golangci-lint install pinned to the same version as
`.github/workflows/go.yml`, gofumpt install, govulncheck install,
Claude Code CLI install (the `claude.ai/install.sh` curl, guarded
by `command -v claude`), `make install-hooks`,
`go install ./cmd/aiwf` + `aiwf init`, env-gated Playwright
install keyed on `AIWF_DEVCONTAINER_E2E`, and the final banner
block containing the literal strings "23min/ai-workflow-rituals",
"PROJECT scope", "aiwf-extensions", and "wf-rituals".
**Edge cases**: any block present but not idempotent (no
`command -v` guard or equivalent re-run safety) fails; banner
block missing any of the four literal strings fails; the
golangci-lint version literal in `init.sh` not matching the one
in `.github/workflows/go.yml` (which is the CI source of truth)
fails the cross-file consistency check; env-gating syntax that
doesn't default to `false` when unset fails (would silently
turn the Playwright install on for everyone).
**Code references**: assertion at
`internal/policies/devcontainer_init_script_test.go` (new);
reads the file, asserts each named section via anchored regex,
cross-validates the golangci-lint version against
`.github/workflows/go.yml`.

### AC-5 — .devcontainer/README.md ships operator-facing usage with named sections

`.devcontainer/README.md` is the in-repo operator documentation
for the devcontainer. A future contributor opening the
`.devcontainer/` directory should find this file first and have
enough context to use the container without reading CLAUDE.md.
**Pass criterion**: markdown file present; parsed via the
project's existing markdown-AST walker (the same helper
`internal/policies/` uses elsewhere for markdown structural
checks); contains H2 sections titled `Build`,
`Reopen in Container`, `Environment variables`, and
`Recovery prompt` (or close-equivalent canonical names — the
assertion matches against the canonical set declared in the test);
each section has non-empty body content.
**Edge cases**: any of the four named sections absent fails;
section present but with empty body content fails per the
`entity-body-empty` analog; substring-flat assertion is rejected
per CLAUDE.md's "substring assertions are not structural
assertions" rule — the test must walk the heading hierarchy
rather than grep over the whole file.
**Code references**: assertion at
`internal/policies/devcontainer_readme_shape_test.go` (new);
markdown-AST walk via the existing project parser. If a shared
helper for "assert these H2 sections exist with non-empty
bodies" already exists in `internal/policies/`, reuse it;
otherwise extract one in the same commit.

### AC-6 — CLAUDE.md Operator setup gains Devcontainer subsection with shadow-mount note

CLAUDE.md's existing `## Operator setup` section documents how
operators install the rituals plugins on the host. This AC adds a
subsection that documents the devcontainer-on-macOS additional
step (the shadow-mount), so a reader following the operator-setup
instructions inside a devcontainer finds the relevant guidance.
**Pass criterion**: parse CLAUDE.md; find the `## Operator setup`
section; assert a `### Devcontainer` (or equivalent-named)
subsection exists within it; assert the subsection body contains
the literal URL
`https://github.com/anthropics/claude-code/issues/31388`; assert
the subsection body contains the literal phrase "shadow-mount" or
"plugin index shadow".
**Edge cases**: subsection present in CLAUDE.md but outside the
`## Operator setup` parent fails (structural scoping per
CLAUDE.md's anti-pattern); URL present in CLAUDE.md but outside
the subsection fails; subsection title close but not canonical
(e.g., `### Dev container` with a space) acceptable iff the test
declares the canonical set; missing the workaround-cleanup
guidance (the "remove once #31388 ships" framing) is acceptable
in this AC — that's covered by the inline comment in
`initialize.sh` per AC-3.
**Code references**: assertion at
`internal/policies/claude_md_devcontainer_section_test.go` (new);
markdown-AST walk asserting the subsection's parent is the
expected H2.

### AC-7 — devcontainer-build-smoke.sh exists with build + Go-version check

A smoke script lives in `scripts/` that exercises the container
image build end-to-end. Currently operator-run (no Docker in CI);
CI integration is deferred to a sibling milestone under this
epic. The script's existence + executability + structural shape
is the mechanical assertion this AC pins.
**Pass criterion**: `scripts/devcontainer-build-smoke.sh` exists
with mode `0755`; runs
`devcontainer build --workspace-folder ...` against the repo; on
success, runs the built image to invoke `go version` and greps
for `go1.25`; exits 0 on success, non-zero with a diagnostic on
failure; documents in its header comment that it is "operator-run
today, CI-integration pending sibling milestone."
**Edge cases**: script present but missing the version-grep step
fails (we'd be asserting "build succeeded" without verifying the
right Go landed); script missing the header comment about
operator-run-status fails; running the script on a host without
the `devcontainer` CLI installed should exit non-zero with a
clear error pointing at `npm install -g @devcontainers/cli`
(asserted via dry-read of the script's preflight section).
**Code references**: structural assertion at
`internal/policies/devcontainer_smoke_scripts_test.go` (new);
asserts file presence, mode, and the key shell-line patterns
(`devcontainer build`, `go version`, `go1.25`, the preflight
check for `devcontainer` CLI presence). The script itself lives
at `scripts/devcontainer-build-smoke.sh`.

### AC-8 — devcontainer-ci-smoke.sh exists with make-ci-inside-container check

The end-to-end smoke for "the dev loop actually works in the
container." Same operator-run posture as AC-7 today; CI
integration deferred to a sibling milestone under this epic.
**Pass criterion**: `scripts/devcontainer-ci-smoke.sh` exists
with mode `0755`; uses `devcontainer up` +
`devcontainer exec` against this repo; runs `make ci` inside;
exits 0 only if all `make ci` sub-targets (vet, lint, test-race,
coverage, selfcheck) exit 0; non-zero with a captured stderr
tail on failure; documents the operator-run-status in its
header comment.
**Edge cases**: script that runs `make ci` but doesn't
propagate the exit code fails the "green means green"
requirement; missing header comment about operator-run-status
fails; running on a host without Docker daemon should exit
non-zero with a clear error (asserted via dry-read of the
script's preflight section); failure on any sub-target
(specifically `selfcheck`, which depends on aiwf binary install
inside the container per init.sh step 6) must propagate as a
non-zero exit, not be silently swallowed.
**Code references**: structural assertion at
`internal/policies/devcontainer_smoke_scripts_test.go` (shared
file with AC-7); asserts file presence, mode, and the key
shell-line patterns (`devcontainer exec`, `make ci`, and exit
code propagation via `set -e` or explicit `exit "$?"`). Script
lives at `scripts/devcontainer-ci-smoke.sh`.

## First-boot recovery

Anticipated failure modes for the first "Reopen in Container"
attempt. A clean Claude session in the container reads this
section to act on *"this happened, fix it"* prompts. **When a
new failure mode surfaces, add an entry here before fixing**,
so the next failure of the same shape is one-shot.

- **Image build fails on `devcontainer-lock.json` SHA
  mismatch.** Symptom: `devcontainer build` errors with
  "feature SHA does not match lock". Fix: delete
  `devcontainer-lock.json` and rebuild — it regenerates from
  `features:` declarations. Commit the new lock file.
- **`initializeCommand` symlinks already exist with wrong
  target.** Symptom: `ln -sfn` succeeds but mount points to a
  stale directory. Fix:
  `rm /tmp/.claude-mount /tmp/.claude-plugins-mount /tmp/.gh-mount`
  on host, then "Rebuild Container."
- **`postCreateCommand` fails on golangci-lint install
  (network or version drift).** Symptom: install step errors;
  container is up but unusable. Fix: re-run
  `bash .devcontainer/init.sh` inside the container; the
  script is idempotent. If the v2.11.4 release is gone from
  the install endpoint, bump the pin in `init.sh` to match
  CI's current pin (`.github/workflows/go.yml`).
- **`aiwf doctor` still warns after plugin install.** Symptom:
  both plugins appear in `/plugin list` but `aiwf doctor`
  keeps reporting `recommended-plugin-not-installed`.
  Diagnosis: plugins installed at USER scope instead of
  PROJECT scope (the CLI form defaults to USER). Fix:
  uninstall via `/plugin`, re-install via the interactive
  `/plugin` menu choosing PROJECT scope explicitly.
- **`git push` from inside the container prompts for
  credentials.** Symptom: HTTPS push challenges for a
  username/token. Diagnosis: gh credential helper config
  didn't apply (init.sh failed or was re-run before
  `~/.config/gh` mount populated). Fix: confirm the mount
  via `ls /home/vscode/.config/gh`; re-run the credential
  helper config block from `init.sh`; `gh auth status` should
  show the host token.
- **Workspace mount missing siblings.** Symptom:
  `ls /workspaces` shows only `aiwf/`, not the sibling
  repos. Diagnosis: `workspaceMount` resolved against
  `${localWorkspaceFolder}` rather than
  `${localWorkspaceFolder}/..`. Fix: confirm the `..` in
  `devcontainer.json`'s `workspaceMount` source string;
  rebuild.
- **Shadow-mount conflict (host plugin index corrupted by
  container write).** Symptom: after container session,
  host's `~/.claude/plugins/<index>` has Linux paths and
  Claude on the host can't find plugins. Diagnosis:
  `~/.claude/plugins` mount didn't shadow correctly
  (initializeCommand symlink wrong, or the `mounts:` order
  in `devcontainer.json` didn't override). Fix: on host,
  restore `~/.claude/plugins` from `~/.claude-linux/plugins`'s
  inverse, or re-run `/plugin` on host. Long-term fix is
  upstream via claude-code#31388.
- **`fatal: not a git repository: (null)` on first
  `git config --global` call.** Symptom: init.sh dies
  immediately at "Configuring git identity"; even
  `git config --global` (which shouldn't need a repo) fails.
  Diagnosis: stray `GIT_DIR`, `GIT_WORK_TREE`, or
  `GIT_COMMON_DIR` in the container env, set by VS Code's
  dev-containers extension's git probe. Fix: `unset GIT_DIR
  GIT_WORK_TREE GIT_COMMON_DIR` at the top of init.sh
  (already in place since commit `ba2abe5e`). This entry
  documents the failure shape so the next clean Claude
  session diagnosing it has a one-shot answer.
- **`aiwf init: creating hooks dir: mkdir /Users: permission
  denied`.** Symptom: aiwf init step of postcreate fails on
  mkdir of `/Users`. Diagnosis: stale absolute
  `core.hooksPath = /Users/.../<repo>/.git/hooks` in the
  repo's `.git/config` (legacy from pre-G38 install-hooks;
  redundant with git's default in any case). Inside the
  container the absolute host path can't be created. Fix:
  defensive `git config --unset core.hooksPath` early in
  init.sh (already in place since commit `00d798f4`). On
  the host this is a no-op (value mirrored git's default).
- **`make install-hooks` → `mkdir: cannot create directory
  '.git': Not a directory`.** Symptom: make install-hooks
  fails on mkdir of `.git/hooks`. Diagnosis: the original
  Makefile target hardcoded `.git/hooks` as a relative
  path, which breaks when `.git` is a worktree pointer
  *file* (not a directory). Fix: Makefile install-hooks
  now uses `HOOKS_DIR=$(git rev-parse --git-path hooks)`
  which returns the common hooks dir correctly for both
  main checkouts and worktrees (already in place since
  commit `00d798f4`).
- **Pre-commit hook fails with `<wrong-context>/aiwf: not
  found` after switching between host and container.**
  Symptom: in the container, `git commit` errors with
  `/Users/.../go/bin/aiwf: not found`. On the host (after
  a container-side aiwf init), it errors with
  `/go/bin/aiwf: not found`. Diagnosis: `aiwf init` bakes
  an absolute path to the current shell's aiwf binary into
  the hooks (`/Users/.../go/bin/aiwf` on host,
  `/go/bin/aiwf` in container). Git fires hooks from the
  common `.git/hooks/` dir for worktrees by default (NOT
  the per-worktree dir aiwf init also writes to), so the
  hook last baked by one environment fails in the other.
  Immediate fix: re-run `aiwf init` from the environment
  you're committing from — it re-bakes the absolute path
  to that environment's aiwf. Recurring tax: a sibling
  gap tracks the structural fix (have aiwf init write a
  hook that probes multiple known paths, or use PATH
  lookup with a deterministic fallback chain).
