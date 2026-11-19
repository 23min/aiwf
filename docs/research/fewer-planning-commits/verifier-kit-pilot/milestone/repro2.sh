# Reproduces both gaps from the pilot template (bash template.sh <dir>), each variant on a fresh copy.
set -u; T=$1; W=$(mktemp -d)
fresh() { rm -rf "$W/w"; cp -r "$T" "$W/w"; cd "$W/w/a"; git remote set-url origin "$W/w/remote.git"; }
run() { echo "\$ $*"; "$@"; echo "exit $?"; }
state() { echo '$ grep -E "^status:|^      status:" work/epics/*/M-0001-*.md'; grep -E '^status:|^      status:' work/epics/*/M-0001-*.md; }
zero() { echo '$ aiwf check | grep -c zero-acs'; aiwf check | grep -c zero-acs; }
echo "aiwf version: $(aiwf version | head -1)"
echo "=== gap A: a met AC on a done milestone, deferred"; fresh
run aiwf promote M-0001 in_progress; run aiwf promote M-0001/AC-1 met; run aiwf promote M-0001 done
run aiwf promote M-0001/AC-1 deferred; state
echo "=== gap A: a met AC on a done milestone, cancelled"; fresh
aiwf promote M-0001 in_progress >/dev/null; aiwf promote M-0001/AC-1 met >/dev/null; aiwf promote M-0001 done >/dev/null
run aiwf cancel M-0001/AC-1; state
echo "=== gap A: a met AC on a cancelled milestone"; fresh
aiwf promote M-0001 in_progress >/dev/null; aiwf promote M-0001/AC-1 met >/dev/null; run aiwf cancel M-0001
run aiwf promote M-0001/AC-1 deferred; state
echo "=== gap B: start with the only AC cancelled, then done"; fresh
run aiwf cancel M-0001/AC-1; run aiwf promote M-0001 in_progress; run aiwf promote M-0001 done; state; zero
echo "=== gap B: start with the only AC deferred"; fresh
run aiwf promote M-0001/AC-1 deferred; run aiwf promote M-0001 in_progress; state
cd /; rm -rf "$W"
