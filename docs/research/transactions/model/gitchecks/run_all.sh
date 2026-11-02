#!/usr/bin/env bash
# Runs every real-git check and writes OUTPUT.txt beside this script.
cd "$(dirname "$0")"
{ echo "# Real-git checks for the transaction model"
  echo "# git $(git --version | cut -d' ' -f3), $(uname -sr), run $(date -u +%Y-%m-%dT%H:%MZ)"
  for s in g1_trailers.sh g2_push.sh g3_merge.sh g4_rebase_empty.sh; do echo; echo "\$ bash $s"; bash "$s" 2>&1; done
} > OUTPUT.txt
rm -rf "${TMPDIR:-/tmp}"/aiwf-gitcheck-*
