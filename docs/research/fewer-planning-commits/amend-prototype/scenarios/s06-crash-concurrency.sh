#!/usr/bin/env bash
# S06: crash mid-amend and concurrency.
#  a: injected failure between commit-tree and update-ref (amend path)
#  b: kill -9 between commit-tree and update-ref, amend path vs new-commit path
#  c: two concurrent verbs (repo lock): wait-then-amend, and lock busy
#  d: a plain git commit lands during the amend window (update-ref CAS)
#  e: a branch is created / a push happens during the amend window
#     (between the eligibility check and update-ref)
. "$(dirname "$0")/lib.sh"
new_env s06; init_repo repo
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
gapq() { aiwf add gap --title "$1" --body "$GB" 2>&1; }
state() { echo "HEAD: $(lg -1)"; echo "status: $(git status --short | tr '\n' ' ')"; echo "temp dirs in .git: $(ls -d .git/aiwf-* 2>/dev/null | tr '\n' ' ')"; }

section "a: injected failure on the amend path"
x gapq "a base"
AIWF_EXPERIMENT_AMEND_FAIL=before-update-ref x aiwf set-priority G-0001 high
state
x aiwf set-priority G-0001 high
state

section "b1: kill -9 during the amend window"
( AIWF_EXPERIMENT_AMEND_DELAY=4s aiwf retitle G-0001 "killed retitle" > "$T/b1.log" 2>&1 & echo $! > "$T/b1.pid"; wait ) &
sleep 1.5; echo "\$ kill -9 (verb pid $(cat "$T/b1.pid"))"; kill -9 "$(cat "$T/b1.pid")"; wait
cat "$T/b1.log"
state
git diff --stat
section "b1: the next verbs after the kill"
x gapq "b1 after kill"
x aiwf retitle G-0001 "killed retitle"
state
x git checkout -- work; git clean -fdq work; rm -rf .git/aiwf-*; git status --short

section "b2: kill -9 during the same window on the new-commit path (amend off)"
( unset AIWF_EXPERIMENT_AMEND; AIWF_EXPERIMENT_AMEND_DELAY=4s aiwf retitle G-0001 "killed retitle 2" > "$T/b2.log" 2>&1 & echo $! > "$T/b2.pid"; wait ) &
sleep 1.5; kill -9 "$(cat "$T/b2.pid")"; wait
cat "$T/b2.log"
state
x git checkout -- work; git clean -fdq work; rm -rf .git/aiwf-*

section "c1: two verbs at once, the first holding the lock 1s (second waits, then amends)"
AIWF_EXPERIMENT_AMEND_DELAY=1s aiwf add gap --title "c1 first" --body "$GB" > "$T/c1a.log" 2>&1 &
sleep 0.3
aiwf add gap --title "c1 second" --body "$GB" > "$T/c1b.log" 2>&1 &
wait
echo "--- first:"; cat "$T/c1a.log"; echo "--- second:"; cat "$T/c1b.log"
state
ops HEAD
section "c2: the first holds the lock 3s (second gets repo-lock-busy)"
AIWF_EXPERIMENT_AMEND_DELAY=3s aiwf add gap --title "c2 first" --body "$GB" > "$T/c2a.log" 2>&1 &
sleep 0.3
aiwf add gap --title "c2 second" --body "$GB" > "$T/c2b.log" 2>&1; echo "[second exit $?]"
wait
echo "--- first:"; cat "$T/c2a.log"; echo "--- second:"; cat "$T/c2b.log"
state

section "d: a plain git commit lands during the amend window"
AIWF_EXPERIMENT_AMEND_DELAY=3s aiwf add gap --title "d planning" --body "$GB" > "$T/d.log" 2>&1 &
sleep 1
echo "d" >> notes.txt; git add notes.txt; git commit -qm "feat: code during window" >/dev/null 2>&1; echo "\$ git commit (code) -> $(git rev-parse --short HEAD)"
wait
cat "$T/d.log"
state
lg -3

section "e1: a branch is created during the amend window"
x gapq "e1 base"
G=$(ls work/gaps | sort | tail -1 | cut -c1-6)
AIWF_EXPERIMENT_AMEND_DELAY=3s aiwf add decision --title "e1 decision in window" --body $'## Question\n\nq?\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.' > "$T/e1.log" 2>&1 &
sleep 1; x git branch snapshot; wait
cat "$T/e1.log"
echo "main: $(lg -1)"; echo "snapshot: $(lg -1 snapshot)"
x git merge -q --no-ff -m "merge snapshot" snapshot
ops HEAD
x aiwf history "$G"
x aiwf check

section "e2: a push happens during the amend window"
x gapq "e2 base"
G=$(ls work/gaps | sort | tail -1 | cut -c1-6)
AIWF_EXPERIMENT_AMEND_DELAY=3s aiwf set-priority "$G" low > "$T/e2.log" 2>&1 &
sleep 1; x git push -q origin main; wait
cat "$T/e2.log"
echo "remote main: $(git --git-dir="$REMOTE" log -1 --format='%h %s' main)"; echo "local main: $(lg -1)"
x git push origin main
