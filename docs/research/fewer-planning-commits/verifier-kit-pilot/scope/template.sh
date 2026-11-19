# Builds the starting world once: a bare remote, clone a on the epic branch with M-0001
# in progress (not yet authorized), and clone b on main, which the "rebase" event uses to move main.
#   bash template.sh <dir>
set -eu; W=${1:?dir}; EB=epic/E-0001-spike-epic
rm -rf "$W"; mkdir -p "$W"; cd "$W"
git init -q --bare -b main remote.git
git clone -q remote.git a 2>/dev/null; cd a
git config user.email peter@example.com; git config user.name Peter
aiwf init >/dev/null 2>&1; git add -A >/dev/null 2>&1; git commit -qm "chore: init" 2>/dev/null || true
aiwf add epic --title "Spike epic" >/dev/null
aiwf add milestone --epic E-0001 --tdd none --title "Spike milestone" >/dev/null
echo "The thing holds." | aiwf add ac M-0001 --title "Something holds" --body-file - >/dev/null
aiwf promote E-0001 active >/dev/null
git push -q -u origin main >/dev/null 2>&1
git checkout -qb $EB
aiwf promote M-0001 in_progress >/dev/null
cd "$W"; git clone -q remote.git b 2>/dev/null
git -C b config user.email carol@example.com; git -C b config user.name Carol
