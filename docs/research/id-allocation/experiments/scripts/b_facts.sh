#!/usr/bin/env bash
# Design B — git facts a ledger ref relies on, each measured on a bare repo over the local transport.
source "$(dirname "$0")/lib.sh"
setup "$X/runs/b-facts"; mkclone alpha; mkclone beta; mkclone hooked hooks
mkcommit() { # mkcommit <clone> <parent-or-empty> <msg>: a commit whose tree is the parent's plus one file
  local c=$1 p=$2 idx blob t
  idx=$(mktemp "$X/out/idx.XXXXXX"); rm -f "$idx"
  if [ -n "$p" ]; then GIT_INDEX_FILE=$idx git -C "$W/$c" read-tree "$p"; fi
  blob=$(echo "$3" | git -C "$W/$c" hash-object -w --stdin)
  GIT_INDEX_FILE=$idx git -C "$W/$c" update-index --add --cacheinfo "100644,$blob,claims/$3"
  t=$(GIT_INDEX_FILE=$idx git -C "$W/$c" write-tree); rm -f "$idx"
  if [ -n "$p" ]; then git -C "$W/$c" commit-tree "$t" -p "$p" -m "$3"; else git -C "$W/$c" commit-tree "$t" -m "$3"; fi
}
echo "== F1: two clients create refs/aiwf/ids from nothing at once (both root commits)"
a=$(mkcommit alpha "" a1); b=$(mkcommit beta "" b1)
show git -C "$W/alpha" push origin "$a:refs/aiwf/ids"
show git -C "$W/beta"  push origin "$b:refs/aiwf/ids"
echo "== F2: beta fetches, builds on the tip, but alpha moved it first (beta has the old tip)"
show git -C "$W/beta" fetch origin "+refs/aiwf/ids:refs/remotes/origin/aiwf-ids"
p=$(git -C "$W/beta" rev-parse refs/remotes/origin/aiwf-ids)
a2=$(mkcommit alpha "$a" a2); show git -C "$W/alpha" push origin "$a2:refs/aiwf/ids"
b2=$(mkcommit beta "$p" b2); show git -C "$W/beta" push origin "$b2:refs/aiwf/ids"
echo "== F3: same, but beta has already fetched alpha's new tip object (non-fast-forward, not fetch-first)"
git -C "$W/beta" fetch -q origin "+refs/aiwf/ids:refs/remotes/origin/aiwf-ids"
show git -C "$W/beta" push origin "$b2:refs/aiwf/ids"
echo "== F4: explicit compare-and-swap with --force-with-lease against a stale expectation, then the right one"
b3=$(mkcommit beta "$a2" b3)
show git -C "$W/beta" push --force-with-lease="refs/aiwf/ids:$p" origin "$b3:refs/aiwf/ids"
show git -C "$W/beta" push --force-with-lease="refs/aiwf/ids:$a2" origin "$b3:refs/aiwf/ids"
echo "== F5: --force overwrites a moved ref unless the host sets receive.denyNonFastForwards"
show git -C "$W/alpha" push --force origin "$a2:refs/aiwf/ids"
git -C "$W/host.git" config receive.denyNonFastForwards true
show git -C "$W/alpha" push --force origin "$a:refs/aiwf/ids"   # rewind a2 -> a1 with --force
echo "== F5b: the same rewind on a branch ref (refs/heads/aiwf/ids) with receive.denyNonFastForwards still set"
git -C "$W/alpha" push -q origin "$a2:refs/heads/aiwf/ids"
show git -C "$W/alpha" push --force origin "$a:refs/heads/aiwf/ids"
git -C "$W/host.git" config --unset receive.denyNonFastForwards
echo "== F6: a fresh clone does not carry refs/aiwf/*; an ordinary branch aiwf/ids is fetched by default"
rm -rf "$W/fresh"; git clone -q "$W/host.git" "$W/fresh"
echo "  host refs:  $(git -C "$W/host.git" for-each-ref --format='%(refname)' | tr '\n' ' ')"
echo "  fresh clone refs: $(git -C "$W/fresh" for-each-ref --format='%(refname)' | tr '\n' ' ')"
echo "== F7: aiwf's pre-push hook on a ledger push from a clone with hooks installed"
git -C "$W/hooked" fetch -q origin "+refs/aiwf/ids:refs/remotes/origin/aiwf-ids"
h=$(mkcommit hooked "$(git -C "$W/hooked" rev-parse refs/remotes/origin/aiwf-ids)" h1)
show git -C "$W/hooked" push origin "$h:refs/aiwf/ids"
show git -C "$W/hooked" push --no-verify origin "$h:refs/aiwf/ids"
echo "== F8: --atomic push of a feature branch and the ledger: one rejected ref rejects both"
git -C "$W/alpha" checkout -q -b feature/A
g=$(cd "$W/alpha"; addgap "atomic gap")
stale=$(git -C "$W/alpha" rev-parse refs/remotes/origin/aiwf-ids 2>/dev/null || echo "$a")
x=$(mkcommit alpha "$a" x1)   # parent is old: non-fast-forward
show git -C "$W/alpha" push --atomic --no-verify origin feature/A "$x:refs/aiwf/ids"
echo "  host has feature/A? $(git -C "$W/host.git" rev-parse -q --verify refs/heads/feature/A || echo no)"
