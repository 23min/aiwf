#!/usr/bin/env bash
# Project hook — runs INSIDE the container after the kit's post-create.sh (once per container).
# Owned by this repo: `copier update` never changes this file.
#
# aiwf's own setup: its pinned Go tools, aiwf built from this checkout (never a release), its
# framework hooks, and the opt-in Playwright install. Idempotent — it runs again after every
# rebuild. Git identity and the gh credential helper come from the kit's post-create, the Go
# version from devcontainer.json's containerEnv.GOTOOLCHAIN; see .devcontainer/project/README.md.
set -euo pipefail

cd "$(dirname "$0")/../.."

# --- core.hooksPath -------------------------------------------------------------------------
# A host path to this repository's own hooks directory disables every git hook here; see the script.
bash .devcontainer/project/hooks-path.sh

# --- Go tooling -----------------------------------------------------------------------------
# golangci-lint and govulncheck must match .github/workflows/go.yml and gitleaks
# .github/workflows/gitleaks.yml, so local checks agree with CI; policy tests in
# internal/policies/ hold those pins.
GOLANGCI_LINT_VERSION="v2.11.4"
if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "==> Installing golangci-lint ${GOLANGCI_LINT_VERSION}"
  curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sh -s -- -b "$(go env GOPATH)/bin" "${GOLANGCI_LINT_VERSION}"
fi

GOFUMPT_VERSION="v0.11.0"
if ! command -v gofumpt >/dev/null 2>&1; then
  echo "==> Installing gofumpt ${GOFUMPT_VERSION}"
  go install "mvdan.cc/gofumpt@${GOFUMPT_VERSION}"
fi

GOIMPORTS_VERSION="v0.49.0"
if ! command -v goimports >/dev/null 2>&1; then
  echo "==> Installing goimports ${GOIMPORTS_VERSION}"
  go install "golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}"
fi

GOVULNCHECK_VERSION="v1.6.0"
if ! command -v govulncheck >/dev/null 2>&1; then
  echo "==> Installing govulncheck ${GOVULNCHECK_VERSION}"
  go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"
fi

# The local pre-push secret scan runs only when gitleaks is installed.
GITLEAKS_VERSION="v8.30.1"
if ! command -v gitleaks >/dev/null 2>&1; then
  echo "==> Installing gitleaks ${GITLEAKS_VERSION}"
  go install "github.com/zricethezav/gitleaks/v8@${GITLEAKS_VERSION}"
fi

# --- aiwf from source, and its hooks ----------------------------------------------------------
# The container runs the aiwf this checkout builds. `aiwf init` regenerates the framework hooks
# and the gitignored skill adapters; stdin comes from /dev/null so an undecided consent prompt
# is left undecided instead of waiting while it holds the repo lock (G-0446).
echo "==> Installing aiwf from this checkout and materializing its framework files"
go install ./cmd/aiwf
aiwf init --no-prompt </dev/null || true

# make install-hooks links scripts/git-hooks/pre-commit into the chain aiwf's pre-commit calls.
echo "==> Installing the kernel pre-commit chain"
make install-hooks

# --- Playwright (opt-in) --------------------------------------------------------------------
# Off unless AIWF_DEVCONTAINER_E2E=true is set for the rebuild; only the HTML renderer's
# end-to-end tests need it.
if [[ "${AIWF_DEVCONTAINER_E2E:-false}" == "true" ]]; then
  echo "==> Installing Playwright + Chromium (AIWF_DEVCONTAINER_E2E=true)"
  (cd e2e/playwright && npm install && npx playwright install chromium)
fi
