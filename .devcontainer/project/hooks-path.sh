#!/usr/bin/env bash
# Repairs a core.hooksPath that works on the host but not in the container. Called by
# post-create.sh before aiwf init and make install-hooks; safe to run again.
#
# .git/config is shared with the host. A core.hooksPath that is the host's absolute path to this
# checkout's own hooks directory (<host checkout>/.git/hooks) does not exist in the container, and
# git then runs no hook at all here. When the repository's own config holds exactly that value and
# it is missing here, unsetting it changes nothing on the host — git's default there is that same
# directory — and restores every hook in the container. AIWF_HOST_CHECKOUT carries the host
# checkout's path (devcontainer.json containerEnv). Any other value, or no AIWF_HOST_CHECKOUT, is
# left as it is.
set -euo pipefail

host_checkout=${AIWF_HOST_CHECKOUT:-}
[ -n "$host_checkout" ] || exit 0
hooks_path=$(git config --local --get core.hooksPath || true)
if [ "${hooks_path%/}" = "${host_checkout%/}/.git/hooks" ] && [ ! -d "$hooks_path" ]; then
  git config --local --unset-all core.hooksPath
  echo "==> Unset core.hooksPath=$hooks_path: the host's hooks directory for this checkout, missing here"
fi
