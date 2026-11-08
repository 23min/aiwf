#!/usr/bin/env bash
# Experiment 2 fixture: side-ref verbs between code commits.   setup2.sh <run dir>
# Leaves <run dir>/a on epic/E-0001-spike-epic with two side-recorded verbs pending
# (M-0001 promoted, G-0002 added) and one code commit on the branch; branch `other`
# (from main) has a retitle of M-0001 made by a second clone; `unrelated` touches only code.
set -euo pipefail
. "$(dirname "$0")/lib.sh" "$1"
fresh_home
B=epic/E-0001-spike-epic
cd "$W"; git init -q --bare remote.git
git clone -q remote.git a 2>/dev/null; cd a
aiwf init >/dev/null 2>&1; git add -A; git commit -qm "chore: init" >/dev/null
aiwf add epic --title "Spike epic" >/dev/null
aiwf add milestone --epic E-0001 --tdd none --title "Spike milestone" >/dev/null
echo "The thing holds." | aiwf add ac M-0001 --title "Something holds" --body-file - >/dev/null
printf "## What's missing\n\nA check.\n\n## Why it matters\n\nIt breaks.\n" | aiwf add gap --title "Something is wrong" --body-file - >/dev/null
aiwf promote E-0001 active >/dev/null
git push -q -u origin main 2>/dev/null
BASE=$(git rev-parse HEAD)
git branch unrelated; git checkout -q unrelated; echo u > unrelated.txt; git add unrelated.txt; git commit -qm "feat: unrelated" >/dev/null
git checkout -q main
( cd "$W"; git clone -q remote.git b 2>/dev/null; cd b
  git checkout -qb other; aiwf retitle M-0001 "Spike milestone, renamed elsewhere" >/dev/null; git push -q origin other 2>/dev/null )
git fetch -q origin; git branch -q other origin/other
git checkout -qb $B
echo one > code.txt; git add code.txt; git commit -qm "feat: c1" >/dev/null
sideverb.sh promote M-0001 in_progress
printf "## What's missing\n\nX.\n\n## Why it matters\n\nY.\n" | sideverb.sh add gap --title "Found while coding" --body-file -
echo two >> code.txt; git add code.txt; git commit -qm "feat: c2" >/dev/null
echo "BASE=$BASE" > "$W/shas.env"; echo "B=$B" >> "$W/shas.env"
echo "fixture ready"
