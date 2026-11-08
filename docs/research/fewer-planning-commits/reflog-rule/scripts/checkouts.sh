#!/usr/bin/env bash
# How often the reflog rule's stricter test would bite: checkouts in a clone's HEAD reflog
# that land on a non-merge verb commit (the next verb after such a checkout commits anew).
# Linked worktrees keep their own HEAD reflogs, deleted with the worktree; this counts the
# main checkout only.
# Usage: checkouts.sh <repo>
set -eu
cd "${1:?repo}"
echo "HEAD reflog: $(git reflog HEAD | wc -l) entries, oldest $(git reflog HEAD --date=short --format=%gd | tail -1)"
n=0; v=0
while read -r c; do
  n=$((n+1))
  if [ "$(git rev-list --parents -n1 "$c" | wc -w)" -eq 2 ] && [ -n "$(git log -1 --format='%(trailers:key=aiwf-verb,valueonly)' "$c")" ]; then v=$((v+1)); fi
done < <(git reflog HEAD --format='%H%x09%gs' | awk -F'\t' '$2 ~ /^checkout:/ {print $1}')
echo "checkouts: $n; landing on a non-merge verb commit: $v"
