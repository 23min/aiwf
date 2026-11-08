#!/usr/bin/env bash
# Intermediate states: for each pair in .git/fold-map (old -> re-created), check out
# both in a detached worktree and compare `aiwf check` (full and --since base) and ./test.sh.
#   inter.sh <run dir>
set -uo pipefail
. "$(dirname "$0")/lib.sh" "$1"
cd "$W/a"; . "$W/shas.env"
summ() { # findings as "severity code subject" lines, sorted
  python3 -I -c '
import json,sys
d=json.load(sys.stdin)
for f in d.get("findings",[]):
    print(f.get("severity"), f.get("code"), f.get("entity_id") or f.get("subject") or f.get("path") or "")' | sort | uniq -c | sed 's/^ *//' | tr '\n' ';'; }
run_at() {
  local c=$1 d=$W/wt-$1
  git worktree add -q --detach "$d" "$c" 2>/dev/null
  ( cd "$d"
    full=$(aiwf check --format=json 2>/dev/null | summ)
    since=$(aiwf check --format=json --since "$BASE" 2>/dev/null | summ)
    if [ -x test.sh ]; then ./test.sh >/dev/null 2>&1 && t=pass || t=FAIL; else t=absent; fi
    echo "test=$t"; echo "full=$full"; echo "since=$since"
    echo "status: M-0001=$(grep -h '^status:' work/epics/*/M-0001-*.md | cut -d' ' -f2) G-0001=$(grep -h '^status:' work/gaps/G-0001-*.md | cut -d' ' -f2)" )
  git worktree remove --force "$d"
}
while read -r o n; do
  [ "$n" = "$(git rev-parse HEAD)" ] && continue
  echo "=== $(git log -1 --format='%s' "$o")"
  diff <(run_at "$o") <(run_at "$n") >$W/inter-diff && echo "  identical: $(run_at "$o" | tr '\n' ' ')" || { echo "  original:"; run_at "$o" | sed 's/^/    /'; echo "  re-created:"; run_at "$n" | sed 's/^/    /'; }
done < .git/fold-map
rm -f $W/inter-diff
