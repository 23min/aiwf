#!/usr/bin/env bash
# Forks made from pre-fold commits, used after the fold.   forks.sh <src run> <run dir>
set -uo pipefail
SRC=$1
"$(dirname "$0")/clone-run.sh" "$SRC" "$2"
. "$(dirname "$0")/lib.sh" "$2"
cd "$W/a"; . "$W/shas.env"
opcount() { # how many commits on HEAD's history record each operation (by verb+entity+to)
  git log --format='%H%x1f%(trailers:key=aiwf-verb,valueonly,separator=|)%x1f%(trailers:key=aiwf-entity,valueonly,separator=|)%x1f%(trailers:key=aiwf-to,valueonly,separator=|)' "$BASE..HEAD" |
  python3 -I -c '
import sys,collections
c=collections.Counter()
for line in sys.stdin:
    h,v,e,t=line.rstrip("\n").split("\x1f")
    vs,es,ts=v.split("|"),e.split("|"),t.split("|")
    if not v: continue
    # one group per verb value, in order; to may be missing for ack
    for i,vv in enumerate(vs):
        c[(vv, es[i] if i<len(es) else "?", ts[i] if i<len(ts) else "")]+=1
for k,n in sorted(c.items()): print("  %dx %s %s %s" % (n,*k))'; }

echo "## fork 1: branch side from pre-fold commit $(git log -1 --format='%h %s' "$C3")"
git branch side "$C3"
echo "## fork 2: worktree wt (branch wt) from pre-fold commit $(git log -1 --format='%h %s' "$V4")"
git worktree add -q -b wt "$W/wt" "$V4"
( cd "$W/wt"; echo w > wt.txt; git add wt.txt; git commit -qm "feat: work in second worktree"
  printf "## What's missing\n\nX.\n\n## Why it matters\n\nY.\n" | aiwf add gap --title "Found in worktree" --body-file - >/dev/null
  echo "wt head: $(git log -1 --format='%h %s')" )
git checkout -q side; echo s > side.txt; git add side.txt; git commit -qm "feat: side work"; git checkout -q "$B"
echo "## fork 3: stash on the pre-fold tip"
echo six >> code.txt; git stash -q; echo "stash base: $(git log -1 --format='%h %s' 'stash@{0}^1')"

echo "## fold"; fold.sh "$BASE"
echo "## operation records on HEAD after fold"; opcount

echo "## git merge side"
git merge --no-edit side 2>&1 | tail -n 4; echo "merge exit ${PIPESTATUS[0]}"
if [ -n "$(git diff --name-only --diff-filter=U)" ]; then
  echo "conflicted: $(git diff --name-only --diff-filter=U | tr '\n' ' ')"; git diff | head -20
  echo "-- resolve by taking ours for each conflicted file, then commit"
  git checkout --ours -- $(git diff --name-only --diff-filter=U); git add -A; git commit -q --no-edit; echo "merge committed: $(git log -1 --format='%h %s')"
fi
echo "## operation records after merging side"; opcount
echo "## aiwf check (full)"; aiwf check 2>&1 | grep -E '^error|findings \('
echo "## aiwf check --since base"; aiwf check --since "$BASE" 2>&1 | grep -E '^error|findings \('
echo "## aiwf history M-0001"; aiwf history M-0001 2>&1 | grep -vE '^\s{4}' | cut -c1-140

echo "## git merge wt"
git merge --no-edit wt 2>&1 | tail -n 4; echo "merge exit ${PIPESTATUS[0]}"
if [ -n "$(git diff --name-only --diff-filter=U)" ]; then
  echo "conflicted: $(git diff --name-only --diff-filter=U | tr '\n' ' ')"
  git checkout --ours -- $(git diff --name-only --diff-filter=U); git add -A; git commit -q --no-edit; echo "merge committed"
fi
echo "## operation records after merging wt"; opcount
echo "## aiwf check --since base"; aiwf check --since "$BASE" 2>&1 | grep -E '^error|findings \('
echo "## aiwf history G-0001"; aiwf history G-0001 2>&1 | grep -vE '^\s{4}' | cut -c1-140

echo "## git stash apply"
git stash apply 2>&1 | tail -n 3; echo "apply exit ${PIPESTATUS[0]}"; git status --short
git checkout -q -- code.txt
echo "## git stash branch sb (branch at the stash base, a pre-fold commit), commit, merge"
git stash branch sb 2>&1 | tail -n 1; git commit -qam "feat: from stash"; git checkout -q "$B"
git merge --no-edit sb 2>&1 | tail -n 3; echo "merge exit ${PIPESTATUS[0]}"
if [ -n "$(git diff --name-only --diff-filter=U)" ]; then
  echo "conflicted: $(git diff --name-only --diff-filter=U | tr '\n' ' ')"
  git checkout --ours -- $(git diff --name-only --diff-filter=U); git add -A; git commit -q --no-edit; echo "merge committed"
fi
echo "## operation records after merging sb"; opcount
echo "## aiwf check --since base"; aiwf check --since "$BASE" 2>&1 | grep -E '^error|findings \('
echo "## hooked push"; git push -u origin "$B" 2>&1 | grep -E '^error|findings \(|->|rejected'; echo "push exit ${PIPESTATUS[0]}"
