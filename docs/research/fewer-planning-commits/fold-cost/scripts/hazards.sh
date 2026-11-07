#!/usr/bin/env bash
# Experiment 2 hazards, each on a fresh copy of the fixture.   hazards.sh <fixture run> <out prefix>
set -uo pipefail
SRC=$1; PFX=$2
SD="$(cd "$(dirname "$0")" && pwd)"
fresh() { "$SD/clone-run.sh" "$SRC" "$PFX-$1"; . "$SD/lib.sh" "$PFX-$1"; cd "$W/a"; . "$W/shas.env"; git update-index -q --refresh >/dev/null; }
pend() { echo "pending on side ref: $(git log --format=%s "refs/aiwf/pending-base/$B..refs/aiwf/pending/$B" 2>/dev/null | wc -l) ops; status: [$(git status --short | tr '\n' ';')]"; }
chk() { aiwf check --since "$BASE" 2>&1 | grep -E '^error|findings \(' | cut -c1-200; }

echo "################ H1 git status noise"
fresh h1; git status --short; echo "lines: $(git status --short | wc -l)"
echo "-- git diff --stat"; git diff --stat | tail -1

echo "################ H2 git add -A && git commit (a code change)"
fresh h2; echo three >> code.txt; git add -A; git commit -m "feat: c3" 2>&1 | grep -vE '^\s*$' | tail -4; echo "commit exit ${PIPESTATUS[0]}"
echo "files in the code commit: $(git show --name-only --format= HEAD | tr '\n' ' ')"; pend
echo "-- aiwf check --since base"; chk
echo "-- submit"; submit.sh; echo "planning commit touches: [$(git show --name-only --format= HEAD | tr '\n' ' ')]"
echo "-- aiwf check --since base after submit"; chk
echo "-- hooked push (first push, no upstream)"; git push -u origin "$B" 2>&1 | grep -E '^error|findings \(|->|rejected' | cut -c1-160

echo "################ H3 git commit -a (a code change)"
fresh h3; echo three >> code.txt; git commit -qam "feat: c3" 2>&1 | grep -vE 'no findings|^\s*$'; echo "files in the code commit: $(git show --name-only --format= HEAD | tr '\n' ' ')"; pend

echo "################ H3b plain git commit after git add of a code path only"
fresh h3b; echo three >> code.txt; git add code.txt; git commit -qm "feat: c3" 2>&1 | grep -vE 'no findings|^\s*$'; echo "files in the code commit: $(git show --name-only --format= HEAD | tr '\n' ' ')"; pend

echo "################ H4 git merge other (other retitled M-0001, the pending file)"
fresh h4; git merge --no-edit other 2>&1 | tail -4; echo "merge exit ${PIPESTATUS[0]}"; pend
echo "-- git merge unrelated (code only)"; git merge --no-edit unrelated 2>&1 | tail -2; echo "merge exit ${PIPESTATUS[0]}"; pend
echo "-- git pull --rebase origin main"; git pull --rebase origin main 2>&1 | tail -2; echo "exit ${PIPESTATUS[0]}"
echo "-- git rebase main"; git rebase main 2>&1 | tail -2; echo "exit ${PIPESTATUS[0]}"

echo "################ H5 git stash, then a side verb, then stash pop"
fresh h5; git stash 2>&1 | tail -1; pend
echo "-- aiwf show M-0001 after stash: $(aiwf show M-0001 2>&1 | head -1)"
echo "-- sideverb promote M-0001/AC-1 met (pending file stashed away)"; sideverb.sh promote M-0001/AC-1 met 2>&1 | tail -6; echo "exit ${PIPESTATUS[0]}"; pend
echo "-- git stash pop"; git stash pop 2>&1 | tail -3; echo "exit ${PIPESTATUS[0]}"; pend
echo "-- git stash -u then git stash pop"; fresh h5b; git stash -u 2>&1 | tail -1; pend; git stash pop -q 2>&1 | tail -2; pend

echo "################ H6 git checkout unrelated (pending files carried along)"
fresh h6; git checkout unrelated 2>&1 | tail -3; echo "checkout exit ${PIPESTATUS[0]}"
echo "on $(git branch --show-current): aiwf show M-0001: $(aiwf show M-0001 2>&1 | head -1)"; echo "aiwf show G-0002: $(aiwf show G-0002 2>&1 | head -1)"; pend
echo "-- sideverb on unrelated: promote G-0001 addressed --by-commit HEAD"
sideverb.sh promote G-0001 addressed --by-commit "$(git rev-parse HEAD)" 2>&1 | tail -6; echo "exit ${PIPESTATUS[0]}"
echo "side refs now: $(git for-each-ref --format='%(refname)' refs/aiwf/ | tr '\n' ' ')"
echo "-- git checkout other (other changed the pending file)"; fresh h6b; git checkout other 2>&1 | tail -4; echo "checkout exit ${PIPESTATUS[0]}"

echo "################ H7 git worktree add (new branch from the epic branch tip)"
fresh h7; git worktree add -q -b wt "$W/wt" 2>&1; ( cd "$W/wt"; echo "in wt: aiwf show M-0001: $(aiwf show M-0001 2>&1 | head -1)"; echo "in wt: aiwf show G-0002: $(aiwf show G-0002 2>&1 | head -1)" )

echo "################ H8 unmodified aiwf verbs against pending files"
fresh h8
echo "-- aiwf promote M-0001/AC-1 met (pending file M-0001)"; aiwf promote M-0001/AC-1 met 2>&1 | head -6; echo "exit ${PIPESTATUS[0]}"
echo "-- aiwf edit-body G-0002 (pending, untracked)"; aiwf edit-body G-0002 2>&1 | head -4; echo "exit ${PIPESTATUS[0]}"
echo "-- aiwf promote G-0001 addressed --by-commit HEAD (no pending change on G-0001)"; aiwf promote G-0001 addressed --by-commit "$(git rev-parse HEAD)" 2>&1 | tail -2; echo "exit ${PIPESTATUS[0]}"
echo "branch now: $(git log -1 --format='%h %s')"; pend
echo "-- aiwf add gap (allocator vs untracked pending G-0002)"; printf "## What's missing\n\nZ.\n\n## Why it matters\n\nW.\n" | aiwf add gap --title "Another" --body-file - 2>&1 | tail -2; echo "exit ${PIPESTATUS[0]}"; git log -1 --format='%h %s'

echo "################ H9 aiwf check / history / status between verbs"
fresh h9; echo "-- aiwf check (full)"; aiwf check 2>&1 | grep -E '^error|findings \('
echo "-- aiwf check --since base"; chk
echo "-- aiwf history M-0001"; aiwf history M-0001 2>&1 | cut -c1-120
echo "-- aiwf status (in flight)"; aiwf status 2>&1 | head -12

echo "################ H10 git reset --hard, git clean -fd"
fresh h10; git reset -q --hard; pend; echo "aiwf show M-0001: $(aiwf show M-0001 2>&1 | head -1)"
echo "-- next sideverb (promote M-0001/AC-1 met)"; sideverb.sh promote M-0001/AC-1 met 2>&1 | tail -4; echo "exit ${PIPESTATUS[0]}"; pend
fresh h10b; git clean -fdq; pend

echo "################ H11 push between verbs"
fresh h11; git push -u origin "$B" 2>&1 | grep -E '^error|findings \(|->|rejected' | cut -c1-160
echo "remote side refs: [$(git ls-remote origin 'refs/aiwf/*' | wc -l)]"

echo "################ H12 clean run: code commit with explicit paths, more side verbs, merge main, submit"
fresh h12
sideverb.sh promote M-0001/AC-1 met 2>&1 | tail -1
git merge -q --no-edit unrelated; echo "merged unrelated: exit $?"
echo three >> code.txt; git add code.txt; git commit -qm "feat: c3" >/dev/null
submit.sh; git log --format='%h %s' "$BASE..HEAD"
echo "status after submit: [$(git status --short)]"
echo "-- aiwf check --since base"; chk
echo "-- aiwf history M-0001"; aiwf history M-0001 2>&1 | grep -vE '^\s{4}' | cut -c1-120
echo "-- hooked push"; git push -u origin "$B" 2>&1 | grep -E '^error|findings \(|->|rejected' | cut -c1-160
