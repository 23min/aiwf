#!/usr/bin/env bash
# Experiment 2, second battery.   hazards2.sh <fixture run> <out prefix>
set -uo pipefail
SRC=$1; PFX=$2
SD="$(cd "$(dirname "$0")" && pwd)"
fresh() { "$SD/clone-run.sh" "$SRC" "$PFX-$1"; . "$SD/lib.sh" "$PFX-$1"; cd "$W/a"; . "$W/shas.env"; git update-index -q --refresh >/dev/null; }
pend() { echo "pending on side ref: $(git log --format=%s "refs/aiwf/pending-base/$B..refs/aiwf/pending/$B" 2>/dev/null | wc -l) ops; status: [$(git status --short | tr '\n' ';')]"; }
MS=work/epics/E-0001-spike-epic/M-0001-spike-milestone.md; G2=work/gaps/G-0002-found-while-coding.md

echo "################ H13 stash -u, merge other (retitled M-0001), stash pop, submit"
fresh h13; git stash -u -q; git merge -q --no-edit other; echo "merge exit $?"
git stash pop 2>&1 | grep -E 'CONFLICT|error' ; echo "pop exit ${PIPESTATUS[0]}"; pend
git checkout -q --theirs -- "$MS" 2>/dev/null; git reset -q; git stash drop -q 2>/dev/null; echo "kept the pending side of the conflict in the working tree"; pend
echo "-- submit"; submit.sh 2>&1 | tail -6; echo "submit exit ${PIPESTATUS[0]}"

echo "################ H10c recover pending files from the side ref after git reset --hard + git clean -fd"
fresh h10c; git reset -q --hard; git clean -fdq; pend
git diff "refs/aiwf/pending-base/$B" "refs/aiwf/pending/$B" | git apply --3way 2>&1 | tail -2; echo "apply exit ${PIPESTATUS[1]}"
git reset -q; pend; echo "aiwf show M-0001: $(aiwf show M-0001 2>&1 | head -1)"

echo "################ H15 git branch -m (side ref keyed by branch name)"
fresh h15; git branch -m epic/E-0001-renamed; B=epic/E-0001-renamed
echo "side refs: $(git for-each-ref --format='%(refname)' refs/aiwf/ | tr '\n' ' ')"
submit.sh; pend

echo "################ H14 skip-worktree on pending tracked files + info/exclude for new ones"
fresh h14; git update-index --skip-worktree "$MS"; echo "/$G2" >> .git/info/exclude
echo "-- git status"; git status --short; echo "lines: $(git status --short | wc -l)"
echo "-- git add -A; git commit (code change)"; echo three >> code.txt; git add -A 2>&1 | head -3; git commit -qm "feat: c3" 2>&1 | grep -v 'no findings'
echo "files in the code commit: $(git show --name-only --format= HEAD | tr '\n' ' ')"
echo "-- git commit -a (code change)"; echo four >> code.txt; git commit -qam "feat: c4" 2>&1 | grep -v 'no findings'; echo "files: $(git show --name-only --format= HEAD | tr '\n' ' ')"
echo "pending M-0001 still on disk: $(grep -c 'status: in_progress' "$MS")"
echo "-- git stash (pending file not stashed?)"; echo five >> code.txt; git stash -q; echo "after stash, M-0001 on disk in_progress: $(grep -c 'status: in_progress' "$MS")"; git stash pop -q
git checkout -q -- code.txt
echo "-- git merge other (other retitled M-0001)"; git merge --no-edit other 2>&1 | tail -3; echo "merge exit ${PIPESTATUS[0]}"
echo "after merge: M-0001 on disk: $(grep -E '^(title|status):' "$MS" | tr '\n' ' ')"
echo "skip-worktree bit still set: $(git ls-files -v "$MS" | cut -c1)"
fresh h14b; git update-index --skip-worktree "$MS"; echo "/$G2" >> .git/info/exclude
echo "-- git checkout other (other changed M-0001)"; git checkout other 2>&1 | tail -3; echo "checkout exit ${PIPESTATUS[0]}"
echo "on $(git branch --show-current): M-0001 on disk: $(grep -E '^(title|status):' "$MS" | tr '\n' ' ')"
git checkout -q "$B" 2>&1 | tail -2; echo "back on $(git branch --show-current): M-0001 on disk: $(grep -E '^(title|status):' "$MS" | tr '\n' ' ')"
fresh h14c; git update-index --skip-worktree "$MS"; echo "/$G2" >> .git/info/exclude
echo "-- git reset --hard"; git reset -q --hard; echo "M-0001 on disk after reset --hard: $(grep -E '^status:' "$MS")"
echo "-- git pull --rebase / git rebase main"; git rebase main 2>&1 | tail -1; echo "rebase exit ${PIPESTATUS[0]}"; echo "M-0001 on disk: $(grep -E '^status:' "$MS")"
fresh h14d; git update-index --skip-worktree "$MS"; echo "/$G2" >> .git/info/exclude
echo "-- unmodified aiwf promote M-0001/AC-1 met"; aiwf promote M-0001/AC-1 met 2>&1 | head -2; echo "exit ${PIPESTATUS[0]}"
echo "-- sideverb promote M-0001/AC-1 met"; sideverb.sh promote M-0001/AC-1 met 2>&1 | tail -1; echo "skip-worktree bit after sideverb: $(git ls-files -v "$MS" | cut -c1)"
echo "-- git worktree add"; git worktree add -q -b wt2 "$W/wt2"; echo "in wt2: $(cd "$W/wt2" && aiwf show M-0001 2>&1 | head -1)"
