#!/usr/bin/env bash
# Project hook — runs on the HOST after the kit's initialize.sh (every build/start).
# Owned by this repo: `copier update` never changes this file.
#
# Use for host-side preparation the kit doesn't cover, e.g. checking that a key file exists
# before a mount needs it, and failing loudly if it doesn't. macOS bash 3.2.
set -euo pipefail

# A core.hooksPath that only works on the host leaves every git hook dead in the container; the
# script removes it where that provably changes nothing on the host. It never fails the start.
bash .devcontainer/project/hooks-path.sh
