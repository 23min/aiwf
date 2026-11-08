#!/usr/bin/env bash
# Tags, notes, reflog, post-rewrite, bisect, SHAs in messages.   misc.sh <src run> <run dir>
set -uo pipefail
"$(dirname "$0")/clone-run.sh" "$1" "$2"
. "$(dirname "$0")/lib.sh" "$2"
cd "$W/a"; . "$W/shas.env"
git tag rc1 "$C3"; git tag -a v0.1 -m "annotated release candidate" "$C4"
git notes add -m "reviewed by carol" "$C2"
git config notes.rewriteRef refs/notes/commits
printf '#!/bin/sh\necho "post-rewrite $1" >> "$(git rev-parse --git-dir)/post-rewrite.log"\n' > .git/hooks/post-rewrite; chmod +x .git/hooks/post-rewrite
echo "before fold: describe HEAD = $(git describe --tags HEAD 2>&1)"
fold.sh "$BASE"
newC2=$(awk -v o="$C2" '$1==o{print $2}' .git/fold-map); newC3=$(awk -v o="$C3" '$1==o{print $2}' .git/fold-map); newC4=$(awk -v o="$C4" '$1==o{print $2}' .git/fold-map)
echo "## tags"
echo "rc1 -> $(git rev-parse --short 'rc1^{commit}') (old C3), contained in HEAD: $(git merge-base --is-ancestor rc1 HEAD && echo yes || echo NO)"
echo "v0.1 -> $(git rev-parse --short 'v0.1^{commit}') (old C4), contained in HEAD: $(git merge-base --is-ancestor v0.1 HEAD && echo yes || echo NO)"
echo "git tag --contains <re-created C3>: [$(git tag --contains "$newC3" | tr '\n' ' ')]"
echo "git describe --tags HEAD: $(git describe --tags HEAD 2>&1)"
echo "planning per-verb commits reachable from v0.1: $(git log --format=%s "$BASE..v0.1" | grep -c '^aiwf ')"
echo "## push the folded branch (hooked), then push tag v0.1 (hooked)"
git push -q -u origin "$B" 2>&1 | grep -E '^error|rejected|findings \('
echo "## SHAs in messages, from a fresh clone of the remote"
( cd "$W"; rm -rf fresh; git clone -q --no-local -b "$B" "file://$W/remote.git" fresh 2>/dev/null; cd fresh
  msgsha=$(git log --format=%s | grep -oE '[0-9a-f]{40}' | head -1)
  echo "C3's message names $msgsha: cat-file -> $(git cat-file -t "$msgsha" 2>&1)"
  abc=$(grep -h -A2 addressed_by_commit work/gaps/G-0001-*.md | grep -oE '[0-9a-f]{40}' | head -1)
  echo "G-0001 addressed_by_commit $abc: cat-file -> $(git cat-file -t "$abc" 2>&1)"
  echo "aiwf show G-0001 (lines naming the commit):"; aiwf show G-0001 2>&1 | head -30 )
git push origin v0.1 2>&1 | grep -E 'refusing|^error|->|rejected'; echo "tag push exit ${PIPESTATUS[0]}"
echo "-- same tag push with --no-verify, to see what the remote then holds (scratch bare remote only)"
git push -q --no-verify origin v0.1; echo "per-verb planning commits now on the remote, reachable from the tag: $(git --git-dir="$W/remote.git" log --format=%s "$BASE..v0.1" | grep -c '^aiwf ')"
echo "## notes"
echo "note on old C2: $(git notes show "$C2" 2>&1)"
echo "note on re-created C2: $(git notes show "$newC2" 2>&1)"
echo "## post-rewrite hook log: [$(cat .git/post-rewrite.log 2>/dev/null)]"
echo "## reflog of $B (newest 2)"; git reflog -n 2 "$B"
echo "ORIG_HEAD: $(git rev-parse -q --verify ORIG_HEAD || echo unset)"
echo "## bisect with SHAs recorded before the fold: good=old C2, bad=HEAD"
git bisect start "$B" "$C2" 2>&1 | head -3
echo "commits in $B ^oldC2: $(git rev-list --count "$B" "^$C2")"; git bisect reset >/dev/null 2>&1
echo "## same, with the re-created C2 as good"
git bisect start "$B" "$newC2" 2>&1 | head -2; echo "commits in $B ^newC2: $(git rev-list --count "$B" "^$newC2")"; git bisect reset >/dev/null 2>&1
