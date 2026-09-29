#!/usr/bin/env bash
# Repairs a core.hooksPath that works on the host but not in the container. Called by
# post-create.sh; safe to run again.
#
# .git/config is shared with the host. A core.hooksPath naming the host's absolute path to this
# repository's own hooks directory (e.g. /Users/<me>/Projects/aiwf/.git/hooks) does not exist in
# the container, and git then runs no hook at all here. When the path is missing here and is that
# default directory spelled absolutely, unsetting it changes nothing on the host — git's default
# is that same directory — and restores every hook in the container. Any other value is left as
# it is: a relative path, a path that exists, or one naming a directory of its own.
set -euo pipefail

hooks_path=$(git config --get core.hooksPath || true)
case "$hooks_path" in
  /*/.git/hooks)
    if [ ! -d "$hooks_path" ]; then
      git config --unset core.hooksPath
      echo "==> Unset core.hooksPath=$hooks_path: a host path to this repository's own hooks directory, missing here"
    fi
    ;;
esac
