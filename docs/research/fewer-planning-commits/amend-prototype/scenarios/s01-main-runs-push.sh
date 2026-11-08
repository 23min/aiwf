#!/usr/bin/env bash
# S01: runs of verbs on main, push, run again, push. Remote history,
# aiwf check, the pre-push hook.
. "$(dirname "$0")/lib.sh"
new_env s01; init_repo repo
GAPBODY=$'## What\'s missing\n\nsomething.\n\n## Why it matters\n\nbecause.'
DBODY=$'## Question\n\nq?\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.'

section "run 1: five verbs on main (unpushed)"
x aiwf add epic --title "Epic one"
x aiwf add gap --title "A gap found" --body "$GAPBODY"
x aiwf set-priority G-0001 high
x aiwf add decision --title "Use plain files" --body "$DBODY"
x aiwf promote D-0001 accepted
lg

section "push 1 (pre-push hook runs aiwf check)"
x git push origin main
x git log --format='%h %s' origin/main

section "run 2 after push: first verb must start a new commit, second amends it"
x aiwf retitle G-0001 "A gap found, retitled"
x aiwf promote E-0001 active
lg -3

section "code commit, then run 3"
echo hello > code.txt; git add code.txt; echo "\$ git commit -m 'feat: code'"; git commit -qm "feat: code"
x aiwf promote G-0001 addressed --by D-0001
x aiwf add gap --title "Second gap" --body "$GAPBODY"
lg -4

section "push 2"
x git push origin main
section "remote history"
git --git-dir="$REMOTE" log --format='%h %s' main
echo "commits carrying aiwf-verb on remote: $(git --git-dir="$REMOTE" log --format='%(trailers:key=aiwf-verb,valueonly,separator=%x2C)' main | grep -c .); operations recorded: $(git --git-dir="$REMOTE" log --format='%(trailers:key=aiwf-verb,valueonly)' main | grep -c .)"

section "aiwf check (full)"
x aiwf check
section "message of the run-1 commit"
git log -1 --format=%B "$(git log --format=%H --grep='^aiwf: 5 operations' | head -1)"
section "reflog"
git reflog -12 --format='%h %gs'
