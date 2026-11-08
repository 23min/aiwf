#!/usr/bin/env bash
# Experiment 1 fixture: a branch with aiwf verbs interleaved with code commits.
#   setup1.sh <run dir> [sign]
# AGENT=0 skips the authorize and runs the AC promote as the human.
# Leaves <run dir>/a on branch epic/E-0001-spike-epic, unpushed, and writes the SHAs
# of the interesting commits to <run dir>/shas.env.
set -euo pipefail
. "$(dirname "$0")/lib.sh" "$1"
fresh_home
[ "${2:-}" = sign ] && enable_signing
B=epic/E-0001-spike-epic
cd "$W"; git init -q --bare remote.git
git clone -q remote.git a 2>/dev/null; cd a
aiwf init >/dev/null 2>&1; git add -A; git commit -qm "chore: init"
aiwf add epic --title "Spike epic" >/dev/null
aiwf add milestone --epic E-0001 --tdd none --title "Spike milestone" >/dev/null
echo "The thing holds." | aiwf add ac M-0001 --title "Something holds" --body-file - >/dev/null
printf "## What's missing\n\nA check for the thing.\n\n## Why it matters\n\nThe thing breaks silently.\n" | aiwf add gap --title "Something is wrong" --body-file - >/dev/null
aiwf promote E-0001 active >/dev/null
git push -q -u origin main 2>&1 | grep -E '^error|rejected' || true
git checkout -qb $B

printf 'one\n' > code.txt; printf '#!/bin/sh\ngrep -q one code.txt\n' > test.sh; chmod +x test.sh
git add code.txt test.sh; git commit -qm "feat: c1 add code"; C1=$(git rev-parse HEAD)
aiwf promote M-0001 in_progress >/dev/null; V1=$(git rev-parse HEAD)
echo two >> code.txt; git commit -qam "feat: c2"; C2=$(git rev-parse HEAD)
if [ "${AGENT:-1}" = 1 ]; then aiwf authorize M-0001 --to ai/claude --reason "spike delegation" >/dev/null; fi; V2=$(git rev-parse HEAD)
echo three >> code.txt; git commit -qam "fix: c3, follow-up to $C2"; C3=$(git rev-parse HEAD)
if [ "${AGENT:-1}" = 1 ]; then aiwf promote M-0001/AC-1 met --actor ai/claude --principal human/peter >/dev/null; else aiwf promote M-0001/AC-1 met >/dev/null; fi; V3=$(git rev-parse HEAD)
aiwf promote G-0001 addressed --by-commit "$C2" >/dev/null; V4=$(git rev-parse HEAD)
EPIC=$(ls work/epics/E-0001-*/epic.md)
printf '\nA wording tweak made by hand.\n' >> "$EPIC"
git commit -qam "docs: tweak epic wording by hand"; H=$(git rev-parse HEAD)
aiwf acknowledge illegal "$H" --for-entity E-0001 --reason "hand edit of wording, reviewed" >/dev/null; A=$(git rev-parse HEAD)
echo four >> code.txt; git commit -qam "feat: c4"; C4=$(git rev-parse HEAD)
( cd "$W"; git clone -q remote.git b 2>/dev/null; cd b
  git config user.email carol@example.com; git config user.name Carol; git config commit.gpgsign false
  echo y > other.txt; git add other.txt; git commit -qm "feat: other on main"; git push -q origin main 2>/dev/null )
git fetch -q origin
git merge -q --no-ff --no-edit origin/main; M=$(git rev-parse HEAD)
echo five >> code.txt; git commit -qam "feat: c5"; C5=$(git rev-parse HEAD)
BASE=$(git merge-base "$C1" origin/main)
cat > "$W/shas.env" <<EOF
B=$B
BASE=$BASE
C1=$C1
V1=$V1
C2=$C2
V2=$V2
C3=$C3
V3=$V3
V4=$V4
H=$H
A=$A
C4=$C4
M=$M
C5=$C5
EOF
echo "fixture ready: $(git rev-list --count --first-parent "$BASE..HEAD") commits on $B over $(git rev-parse --short "$BASE")"
