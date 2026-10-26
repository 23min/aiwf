#!/usr/bin/env bash
# Design B — can today's aiwf consume a ledger with no code change?
# A ledger commit holds one claim file per claimed id, at the path aiwf's path classifier reads
# (work/gaps/G-NNNN.md). Main holds G-0001; the ledger claims G-0005.
source "$(dirname "$0")/lib.sh"
ledger_commit() { # ledger_commit <clone> <parent-or-empty> <path> <content>: tree = parent's + one file
  local c=$1 p=$2 idx blob t
  idx=$(mktemp "$X/out/idx.XXXXXX"); rm -f "$idx"
  [ -n "$p" ] && GIT_INDEX_FILE=$idx git -C "$W/$c" read-tree "$p"
  blob=$(printf '%s\n' "$4" | git -C "$W/$c" hash-object -w --stdin)
  GIT_INDEX_FILE=$idx git -C "$W/$c" update-index --add --cacheinfo "100644,$blob,$3"
  t=$(GIT_INDEX_FILE=$idx git -C "$W/$c" write-tree); rm -f "$idx"
  if [ -n "$p" ]; then git -C "$W/$c" commit-tree "$t" -p "$p" -m "claim $3"; else git -C "$W/$c" commit-tree "$t" -m "claim $3"; fi
}
setup "$X/runs/b-consume"; mkclone claimer
c=$(ledger_commit claimer "" work/gaps/G-0005.md "claimed-by: claimer")
git -C "$W/claimer" push -q --no-verify origin "$c:refs/heads/aiwf/ids" "$c:refs/aiwf/ids"
echo "== C1: ledger as an ordinary branch (refs/heads/aiwf/ids), default fetch"
mkclone c1; ( cd "$W/c1"; git checkout -q -b feature/x
  echo "  remote-tracking refs: $(git for-each-ref --format='%(refname)' refs/remotes | tr '\n' ' ')"
  echo "  aiwf add gap allocates: $(addgap "c1 gap")" )
echo "== C2: ledger as refs/aiwf/ids, mirrored to the same local name"
git -C "$W/host.git" update-ref -d refs/heads/aiwf/ids
mkclone c2; ( cd "$W/c2"; git fetch -q origin "+refs/aiwf/ids:refs/aiwf/ids"; git checkout -q -b feature/x
  echo "  local refs/aiwf/ids: $(git rev-parse --short refs/aiwf/ids)"
  echo "  aiwf add gap allocates: $(addgap "c2 gap")" )
echo "== C3: ledger as refs/aiwf/ids, fetched into refs/remotes/origin/aiwf-ids by a configured refspec"
mkclone c3; ( cd "$W/c3"; git config --add remote.origin.fetch "+refs/aiwf/ids:refs/remotes/origin/aiwf-ids"; git fetch -q origin
  git checkout -q -b feature/x
  echo "  aiwf add gap allocates: $(addgap "c3 gap")" )
echo "== C4: the claimer itself files the gap it claimed (G-0005)"
( cd "$W/claimer"; git fetch -q origin; git checkout -q -b feature/claimer
  echo "  claimer's aiwf add gap allocates: $(addgap "claimer gap")"
  echo "  aiwf add flags that take an id: $(aiwf add --help </dev/null 2>&1 | grep -oE -- '--(id|number)\b' | tr '\n' ' ')(none means none)" )
for stub in differs copy; do
  echo "== C6-$stub: allocate first, then claim the allocated id with a stub that $stub the entity file"
  setup "$X/runs/b-consume-$stub"; mkclone alpha; mkclone third
  ( cd "$W/alpha"; git checkout -q -b feature/A; g=$(addgap "alpha gap"); f=$(entity_file "$g")
    if [ $stub = copy ]; then content=$(cat "$f"); else content="claimed-by: alpha"; fi
    c=$(ledger_commit alpha "" "$f" "$content")
    git push -q --no-verify origin feature/A "$c:refs/heads/aiwf/ids"; echo "  alpha: $g claimed at $f" )
  ( cd "$W/third"; git fetch -q origin
    show aiwf show G-0002 | head -3
    cite G-0001 "See G-0002 (means:alpha-gap)."
    show aiwf check | grep -E 'G-0002|findings|exit' | cut -c1-200 )
done
