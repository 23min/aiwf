# Builds the starting world once: clone a on the epic branch, E-0001 active, M-0001 draft with
# tdd: none, and M-0001/AC-1 open with a non-empty body (chapter A1), pushed to a bare remote
# so that aiwf check audits the commits the run makes on top of it.
#   bash template.sh <dir>
set -eu; W=${1:?dir}
rm -rf "$W"; mkdir -p "$W"; cd "$W"
git init -q --bare -b main remote.git
git clone -q remote.git a 2>/dev/null; cd a
git config user.email peter@example.com; git config user.name Peter
aiwf init >/dev/null 2>&1; git add -A >/dev/null 2>&1; git commit -qm "chore: init" 2>/dev/null || true
aiwf add epic --title "Pilot epic" >/dev/null
aiwf add milestone --epic E-0001 --tdd none --title "Pilot milestone" >/dev/null
echo "The thing holds." | aiwf add ac M-0001 --title "Something holds" --body-file - >/dev/null
aiwf promote E-0001 active >/dev/null
git push -q -u origin main >/dev/null 2>&1
git checkout -qb epic/E-0001-pilot-epic
git push -q -u origin epic/E-0001-pilot-epic >/dev/null 2>&1
