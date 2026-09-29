#!/usr/bin/env bash
# Runs on the HOST, from project/initialize.sh, before every container build or start; safe to
# run again. macOS bash 3.2.
#
# A core.hooksPath naming this repository's own hooks directory by its absolute host path works on
# the host, but in the container that directory does not exist and git runs no hook at all. It is
# removed only when that changes nothing on the host: every value set anywhere is this repository's
# own hooks directory (<git common dir>/hooks, a trailing slash allowed), and all of them sit in the
# repository's own config, none in the global or system config. Git's default on the host is then
# that same directory. Anything else is left as it is, and nothing here stops a container start.
set -uo pipefail

common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || exit 0
own="${common%/}/hooks"
everywhere=$(git config --get-all core.hooksPath 2>/dev/null) || exit 0
in_repo=$(git config --local --get-all core.hooksPath 2>/dev/null) || exit 0
[ "$everywhere" = "$in_repo" ] || exit 0
while IFS= read -r value; do
  [ "${value%/}" = "$own" ] || exit 0
done <<EOT
$in_repo
EOT
if git config --local --unset-all core.hooksPath; then
  echo "project initialize: unset core.hooksPath=$own, this repository's own hooks directory, which git uses by default; hooks now run in the container too"
else
  echo "project initialize: could not unset core.hooksPath=$own; git hooks will not run in the container until it is unset" >&2
fi
exit 0
