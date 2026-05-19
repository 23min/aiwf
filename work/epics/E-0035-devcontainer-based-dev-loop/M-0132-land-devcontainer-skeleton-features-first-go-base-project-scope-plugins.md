---
id: M-0132
title: Land .devcontainer skeleton (features-first, Go base, project-scope plugins)
status: draft
parent: E-0035
tdd: none
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

- **Base image:** `mcr.microsoft.com/devcontainers/go:1-1.25-bookworm`.
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

## Acceptance criteria

Eight ACs cover the structural shape of the .devcontainer/ files,
the CLAUDE.md operator-setup subsection, and two operator-run
smoke scripts under `scripts/` that exercise image build and
in-container `make ci`. Six are pure structural assertions
mechanized via `internal/policies/`; two are smoke scripts whose
existence + shell-line structure is asserted, with their
runtime green-ness deferred to operator verification (CI
integration is a sibling milestone under this epic). ACs scaffold
below as `### AC-N — <title>` sub-elements; each carries its own
pass criterion, edge cases, and code references.

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
