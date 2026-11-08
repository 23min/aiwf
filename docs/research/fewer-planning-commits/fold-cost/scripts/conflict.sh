#!/usr/bin/env bash
# A non-verb commit whose change is adjacent to a folded verb's change.   conflict.sh <run dir>
set -uo pipefail
. "$(dirname "$0")/lib.sh" "$1"; fresh_home
cd "$W"; git init -q --bare remote.git; git clone -q remote.git a 2>/dev/null; cd a
aiwf init >/dev/null 2>&1; git add -A; git commit -qm "chore: init" >/dev/null
aiwf add epic --title "Spike epic" >/dev/null
aiwf add milestone --epic E-0001 --tdd none --title "Spike milestone" >/dev/null
echo "The thing holds." | aiwf add ac M-0001 --title "Something holds" --body-file - >/dev/null
git push -q -u origin main 2>/dev/null; BASE=$(git rev-parse HEAD); git checkout -qb work
echo one > code.txt; git add code.txt; git commit -qm "feat: c1" >/dev/null
echo "Another thing holds." | aiwf add ac M-0001 --title "Another holds" --body-file - >/dev/null
MS=$(ls work/epics/E-0001-*/M-0001-*.md)
sed -i 's/^Another thing holds\.$/Another thing holds, and here is why./' "$MS"
git commit -qam "docs: clarify AC-2 by hand" >/dev/null
echo two >> code.txt; git commit -qam "feat: c2" >/dev/null
git log --format='%h %s' $BASE..HEAD
echo '$ fold.sh $BASE'; before=$(git rev-parse HEAD); fold.sh "$BASE"; echo "fold exit $?"
echo "branch unchanged: $([ "$(git rev-parse HEAD)" = "$before" ] && echo yes || echo NO)"
