#!/usr/bin/env bash
# scripts/devcontainer-ci-smoke.sh
#
# Exercises `make ci` end-to-end inside the devcontainer.
# **Operator-run today**; CI integration (Docker-in-Docker matrix
# run) is deferred to a sibling milestone under E-0035. Pins
# M-0132/AC-8.
#
# Preflight: requires the Docker daemon running and the
# `devcontainer` CLI on $PATH (npm install -g @devcontainers/cli).
#
# Usage from the host:
#   bash scripts/devcontainer-ci-smoke.sh
#
# Exits 0 only if `make ci` is green inside the container. The
# `make ci` chain runs vet + lint + test-race + coverage +
# selfcheck; any sub-target failure propagates as non-zero exit
# via `set -e` and the explicit `exit "$?"` at the end.

set -euo pipefail

if ! command -v devcontainer >/dev/null 2>&1; then
  echo "FAIL: devcontainer CLI not found. Install via: npm install -g @devcontainers/cli" >&2
  exit 1
fi

if ! docker info >/dev/null 2>&1; then
  echo "FAIL: Docker daemon not reachable. Start Docker Desktop (or your OCI runtime) and retry." >&2
  exit 1
fi

repo_root="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> devcontainer up (first-time may take several minutes)"
devcontainer up --workspace-folder "${repo_root}"

echo "==> devcontainer exec -- make ci"
devcontainer exec --workspace-folder "${repo_root}" -- make ci
exit "$?"
