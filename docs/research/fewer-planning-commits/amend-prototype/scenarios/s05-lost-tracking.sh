#!/usr/bin/env bash
# S05: a pushed commit that no remote-tracking ref contains.
#  a: --single-branch clone (main, then a pushed second branch)
#  b: git push <url>
#  c: lost push reply (tracking ref left at its pre-push value), then
#     pull --no-rebase and, separately, pull --rebase
. "$(dirname "$0")/lib.sh"
new_env s05; init_repo repo
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
gap() { x aiwf add gap --title "$1" --body "$GB"; G=$(ls work/gaps | sort | tail -1 | cut -c1-6); }
dec() { x aiwf add decision --title "$1" --body $'## Question\n\nq?\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.'; }

section "a: --single-branch clone"
x git clone -q --single-branch --branch main "$REMOTE" "$T/single"
cd "$T/single"
x git config remote.origin.fetch
gap "single main 1"
x git push -q origin main
echo "refs containing HEAD after push: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
gap "single main 2"
lg -2
section "a: second branch pushed from the single-branch clone"
x git switch -q -c epic/E-0009-x
gap "single branch 1"
x git push -q -u origin epic/E-0009-x
echo "refs containing HEAD after push: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
echo "remote has: $(git --git-dir="$REMOTE" log -1 --format='%h %s' epic/E-0009-x)"
x aiwf set-priority "$G" high
lg -2
x git push origin epic/E-0009-x
x git switch -q main

section "b: git push <url>"
cd "$T/repo"; x git pull -q --no-rebase origin main
gap "url push 1"; GURL=$G
x git push -q "$REMOTE" main
echo "refs containing HEAD after push: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
dec "url push decision"
lg -2
x git push origin main
section "b: pull (merge) to get past the rejection"
x git pull --no-rebase origin main
lg --graph -4
ops HEAD
x aiwf history "$GURL"
x aiwf check
x git push origin main
echo "remote main has: $(git --git-dir="$REMOTE" log --oneline main | wc -l) commits"

section "c1: lost push reply, then pull --no-rebase"
OLD=$(git rev-parse origin/main)
gap "lost reply 1"
x git push -q origin main
x git update-ref refs/remotes/origin/main "$OLD"   # the reply never arrived
echo "remote main: $(git --git-dir="$REMOTE" rev-parse --short main); local tracking: $(git rev-parse --short origin/main); HEAD: $(git rev-parse --short HEAD)"
dec "lost reply decision"
lg -2
x git push origin main
x git pull --no-rebase origin main
lg --graph -4
ops HEAD
x aiwf check
x git push origin main

section "c2: lost push reply, then pull --rebase"
OLD=$(git rev-parse origin/main)
gap "lost reply 2"
x git push -q origin main
x git update-ref refs/remotes/origin/main "$OLD"
dec "lost reply decision 2"
lg -2
x git push origin main
x git pull --rebase origin main
lg -3
echo "--- files changed by HEAD:"; git show --stat --format='%s' HEAD | tail -4
ops HEAD
x aiwf history "$G"
x aiwf check
x git push origin main

section "c3: remote-tracking ref deleted after a push (e.g. a bad prune), then the next verb"
gap "deleted ref 1"
x git push -q origin main
x git update-ref -d refs/remotes/origin/main
echo "refs containing HEAD: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
dec "deleted ref decision"
x git push origin main
