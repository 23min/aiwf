#!/usr/bin/env bash
# Runs INSIDE the container via devcontainer.json `postCreateCommand`
# and `updateContentCommand`. Idempotent — every step is safe to
# re-run after a container rebuild.
#
# See .devcontainer/README.md for the operator guide and the
# milestone spec at work/epics/E-0035-*/M-0132-* for the per-decision
# rationale (Q1–Q7 of the design conversation).

set -euo pipefail

echo "==> aiwf devcontainer postcreate"

# --- GIT_* env hygiene ---------------------------------------------
# Defensively unset GIT_DIR/GIT_WORK_TREE/GIT_COMMON_DIR so the
# worktree's .git file is the authoritative source. Some
# devcontainer probe paths leave these set to host paths (or empty
# values) that don't resolve in the container, producing
# `fatal: not a git repository: (null)` on later git operations
# — including `git config --global` calls that have no business
# touching repo discovery in the first place.
unset GIT_DIR GIT_WORK_TREE GIT_COMMON_DIR

# --- worktree .git rewrite -----------------------------------------
# When the workspaceFolder is a git worktree of a sibling repo also
# visible under /workspaces/, the worktree's `.git` file holds a
# host-side absolute path (`gitdir: /Users/.../.git/worktrees/<n>`)
# that doesn't resolve inside the container. Rewrite to a relative
# gitdir pointer so it works in both host and container contexts
# (relative paths resolve from .git's directory).
#
# No-op when .git is a directory (main checkout, not a worktree) or
# already relative.
if [ -f .git ]; then
  gitdir_value=$(sed -n 's|^gitdir:[[:space:]]*||p' .git)
  case "$gitdir_value" in
    .* | "" )
      ;;
    */.git/worktrees/* )
      main_and_more="${gitdir_value%/.git/worktrees/*}"
      main_name=$(basename "$main_and_more")
      wt_name="${gitdir_value##*/}"
      relative_gitdir="../${main_name}/.git/worktrees/${wt_name}"
      echo "==> Rewriting worktree .git gitdir to relative path (host+container portable):"
      echo "       ${gitdir_value}"
      echo "    -> ${relative_gitdir}"
      echo "gitdir: ${relative_gitdir}" > .git
      # Also rewrite the reverse pointer (main repo's
      # .git/worktrees/<n>/gitdir → worktree's .git) to a relative
      # path. Position is 4 levels deep under /workspaces/, so 4 ups
      # reach the /workspaces/ parent before stepping into the
      # worktree dir.
      self_dir=$(basename "$PWD")
      rev_file="../${main_name}/.git/worktrees/${wt_name}/gitdir"
      if [ -f "$rev_file" ]; then
        echo "../../../../${self_dir}/.git" > "$rev_file"
      fi
      ;;
  esac
fi

# --- stale core.hooksPath unset ------------------------------------
# Pre-G38 install-hooks invocations could leave an absolute
# `core.hooksPath` in the repo config that mirrors the default
# `<gitdir>/hooks` value but as a host-side absolute path
# (`/Users/.../`). Inside the container that path doesn't exist;
# `aiwf init` and any hook-resolving verb crashes with a mkdir
# permission-denied on `/Users`. Unset defensively so git's
# default `<gitdir>/hooks` discovery (which works correctly across
# host + container with relative .git pointers) kicks in.
git config --unset core.hooksPath 2>/dev/null || true

# --- git config ----------------------------------------------------
# Match host identity so aiwf commit trailers stay consistent across
# host and container (the runtime-derived actor reads the localpart
# of user.email).
echo "==> Configuring git identity"
git config --global user.name "Peter Bruinsma"
git config --global user.email "peter@23min.com"

# --- gh credential helper ------------------------------------------
# The github-cli feature wires `gh auth git-credential` against an
# absolute path that depends on where `gh` was installed; this
# rewrite uses bare `gh` so $PATH resolution finds it. Per
# Liminara's post-create.sh precedent.
echo "==> Configuring gh credential helper"
for host in https://github.com https://gist.github.com; do
  git config --global --unset-all "credential.${host}.helper" 2>/dev/null || true
  git config --global --add "credential.${host}.helper" ""
  git config --global --add "credential.${host}.helper" "!gh auth git-credential"
done

# --- Go tooling: golangci-lint, gofumpt, govulncheck ---------------
# golangci-lint version must match .github/workflows/go.yml so
# local and CI agree (cross-file consistency asserted by
# PolicyM0132InitScript).
GOLANGCI_LINT_VERSION="v2.11.4"
if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "==> Installing golangci-lint ${GOLANGCI_LINT_VERSION}"
  curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin" "${GOLANGCI_LINT_VERSION}"
fi

if ! command -v gofumpt >/dev/null 2>&1; then
  echo "==> Installing gofumpt"
  go install mvdan.cc/gofumpt@latest
fi

if ! command -v govulncheck >/dev/null 2>&1; then
  echo "==> Installing govulncheck"
  go install golang.org/x/vuln/cmd/govulncheck@latest
fi

# --- Claude Code CLI -----------------------------------------------
# The native installer lands `claude` at ~/.local/bin/claude. The
# devcontainer.json PATH already includes ~/.local/bin via the
# common-utils feature defaults.
if ! command -v claude >/dev/null 2>&1; then
  echo "==> Installing Claude Code CLI"
  curl -fsSL https://claude.ai/install.sh | bash
  export PATH="$HOME/.local/bin:$PATH"
fi

# --- aiwf binary + framework hooks ---------------------------------
# `go install ./cmd/aiwf` is idempotent (it overwrites the prior
# binary); `aiwf init` is idempotent (regenerates the chain-aware
# pre-commit hook and the gitignored skill adapters).
#
# postCreateCommand runs with CWD = workspaceFolder (a devcontainer
# spec guarantee), and workspaceFolder is templated to
# /workspaces/${localWorkspaceFolderBasename} in devcontainer.json,
# so all paths below are relative to the opened folder — main
# checkout, worktree, or any other clone path works without edit.
echo "==> Installing aiwf binary and materializing framework hooks"
go install ./cmd/aiwf
export PATH="$(go env GOPATH)/bin:$PATH"
aiwf init || true

# --- kernel pre-commit chain ---------------------------------------
# `make install-hooks` symlinks scripts/git-hooks/pre-commit into
# .git/hooks/pre-commit.local — the chain target invoked by aiwf's
# chain-aware pre-commit. Idempotent (ln -sf).
echo "==> Installing kernel pre-commit chain"
make install-hooks

# --- Playwright (env-gated) ----------------------------------------
# Default off — most contributors aren't touching the HTML renderer.
# Set AIWF_DEVCONTAINER_E2E=true and rebuild the container to opt
# in. Per Q4 of the design conversation.
if [[ "${AIWF_DEVCONTAINER_E2E:-false}" == "true" ]]; then
  echo "==> Installing Playwright + Chromium (AIWF_DEVCONTAINER_E2E=true)"
  (cd e2e/playwright && npm install && npx playwright install chromium)
fi

# --- post-install banner -------------------------------------------
cat <<'BANNER'

================================================================
aiwf devcontainer ready.

One manual step remaining — install the rituals plugins at PROJECT
scope. The CLI form `claude /plugin install ...` defaults to USER
scope (wrong scope; aiwf doctor will warn). Instead, in Claude
Code at this repo's root:

  /plugin marketplace add 23min/ai-workflow-rituals
  /plugin                   # Discover tab; install each at PROJECT scope:
                            #   - aiwf-extensions
                            #   - wf-rituals
  /reload-plugins

Verify with `aiwf doctor`: the recommended-plugin-not-installed
warnings should fall silent once both plugins are project-scope
installed.

Opt-in env vars:
  AIWF_DEVCONTAINER_E2E=true    # Install Playwright + Chromium on next rebuild

See .devcontainer/README.md for the full operator guide.
================================================================

BANNER
