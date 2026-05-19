#!/bin/bash
# Wrap go test's -exec to ad-hoc sign Darwin test binaries before they run.
# Works around a macOS Sonoma 14.8.x syspolicyd bug that segfaults parsing
# unsigned Mach-O code-signing data (G-0128 first layer; G-0133 outer layer).
# No-op on Linux/Windows; the wrapper still exec's the binary.
set -euo pipefail
if [[ "$(uname)" == "Darwin" ]]; then
  codesign --sign - --force "$1" 2>/dev/null || true
fi
exec "$@"
