# Reproduces the two divergences. Needs the pilot template (bash template.sh <dir>).
set -u; T=$1; W=$(mktemp -d); cp -r "$T" "$W/w"; cd "$W/w/a"; git remote set-url origin "$W/w/remote.git"
echo "== aiwf version: $(aiwf version | head -1)"
echo "== gap 1: a met AC on a done milestone"
aiwf promote M-0001 in_progress >/dev/null; aiwf promote M-0001/AC-1 met >/dev/null; aiwf promote M-0001 done >/dev/null
echo '$ aiwf promote M-0001/AC-1 deferred'; aiwf promote M-0001/AC-1 deferred; echo "exit $?"
grep -E '^status:|^      status:' work/epics/*/M-0001-*.md
cd "$W"; rm -rf w; cp -r "$T" w; cd w/a; git remote set-url origin "$W/w/remote.git"
echo "== gap 2: start with the only AC cancelled"
echo '$ aiwf cancel M-0001/AC-1'; aiwf cancel M-0001/AC-1; echo "exit $?"
echo '$ aiwf promote M-0001 in_progress'; aiwf promote M-0001 in_progress; echo "exit $?"
grep -E '^status:|^      status:' work/epics/*/M-0001-*.md
rm -rf "$W"
