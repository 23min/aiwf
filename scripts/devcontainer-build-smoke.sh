#!/usr/bin/env bash
# scripts/devcontainer-build-smoke.sh
#
# Exercises the devcontainer image build end-to-end. **Operator-run
# today**; CI integration (Docker-in-Docker matrix run) is deferred
# to a sibling milestone under E-0035. Pins M-0132/AC-7.
#
# Preflight: requires the `devcontainer` CLI on $PATH. Install via:
#   npm install -g @devcontainers/cli
#
# Usage from the host:
#   bash scripts/devcontainer-build-smoke.sh
#
# Exits 0 on a clean build with the right Go version, non-zero with
# a diagnostic on failure.

set -euo pipefail

if ! command -v devcontainer >/dev/null 2>&1; then
  echo "FAIL: devcontainer CLI not found. Install via: npm install -g @devcontainers/cli" >&2
  exit 1
fi

repo_root="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> devcontainer build (cold cache may take several minutes)"
build_output=$(devcontainer build --workspace-folder "${repo_root}" 2>&1)
build_status=$?
if [[ $build_status -ne 0 ]]; then
  echo "FAIL: devcontainer build exited $build_status" >&2
  echo "$build_output" >&2
  exit "$build_status"
fi

# Verify the built image runs Go 1.25. Extract the image id from the
# build output (devcontainer build prints "imageName": "<id>" in JSON).
image=$(echo "$build_output" | grep -oE '"imageName"\s*:\s*"[^"]+"' | head -1 | sed -E 's/.*"imageName"\s*:\s*"([^"]+)"/\1/')
if [[ -z "$image" ]]; then
  echo "FAIL: could not extract imageName from build output" >&2
  echo "$build_output" >&2
  exit 1
fi

echo "==> Verifying Go version in image ${image}"
go_ver=$(docker run --rm "$image" go version 2>&1)
if ! echo "$go_ver" | grep -q "go1.25"; then
  echo "FAIL: image does not report go1.25 — got: $go_ver" >&2
  exit 1
fi

echo "PASS: image built clean, ${go_ver}"
