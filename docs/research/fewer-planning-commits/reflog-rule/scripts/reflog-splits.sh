#!/usr/bin/env bash
# Where the reflog rule would have refused an amend in a clone's main checkout: a reflog entry
# written by something other than a verb (verbs write an empty message today) leaves HEAD on a
# non-merge verb commit, and the next entry is a verb committing on top of it. Each is a run of
# verbs the reflog rule splits, whether or not the other clauses would have let it amend; the
# commit left at HEAD is printed with its aiwf-verb, since a ritual's commit is never amended anyway.
# Usage: reflog-splits.sh <repo>
set -eu
cd "${1:?repo}"
git reflog HEAD --format='%H%x09%gs' | tac > "${TMPDIR:-/tmp}/reflog-splits.$$"
prev=""; prevmsg=""; n=0
while IFS=$'\t' read -r h msg; do
  if [ -n "$prev" ] && [ -n "$prevmsg" ] && [ -z "$msg" ] \
     && [ "$(git rev-parse "$h^1" 2>/dev/null)" = "$prev" ] \
     && [ "$(git rev-list --parents -n1 "$prev" | wc -w)" -eq 2 ] \
     && [ -n "$(git log -1 --format='%(trailers:key=aiwf-verb,valueonly)' "$prev")" ]; then
    n=$((n+1)); v=$(git log -1 --format='%(trailers:key=aiwf-verb,valueonly)' "$prev")
    echo "  $(git rev-parse --short "$prev") ($v) after '${prevmsg%%:*}', then verb $(git rev-parse --short "$h"): $(git log -1 --format=%s "$h")"
  fi
  prev=$h; prevmsg=$msg
done < "${TMPDIR:-/tmp}/reflog-splits.$$"
rm -f "${TMPDIR:-/tmp}/reflog-splits.$$"
echo "runs the reflog rule splits in this checkout: $n"
