#!/usr/bin/env bash
# S04: git operations between verbs: second worktree (branch, detached),
# stash, reset --soft/--mixed/--hard, rebase, cherry-pick (-x), pull
# (merge and rebase).
. "$(dirname "$0")/lib.sh"
new_env s04; init_repo repo
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
gap() { x aiwf add gap --title "$1" --body "$GB"; G=$(ls work/gaps | sort | tail -1 | cut -c1-6); }
peer_commit() { # a commit from another clone lands on the remote
  ( cd "$T" && rm -rf peer && git clone -q "$REMOTE" peer && cd peer && echo "$1" >> peer.txt && git add peer.txt && git commit -qm "feat: $1" && git push -q origin main >/dev/null 2>&1 && echo "peer pushed: $(git log -1 --format='%h %s')" )
}

section "a1: second worktree on a new branch cut from HEAD"
gap "gap a1"
x git worktree add -q -b side "$T/wt-side"
gap "gap a1 second"
lg -2
section "a1: verbs inside the side worktree"
( cd "$T/wt-side" && x aiwf add decision --title "side decision" --body $'## Question\n\nq?\n\n## Decision\n\nd.\n\n## Reasoning\n\nr.' && x aiwf set-priority D-0001 high && lg -3 && ls -a | grep -i lock; git status --short )
section "a2: detached worktree at HEAD (no ref names it)"
x git worktree add -q --detach "$T/wt-detached" HEAD
gap "gap a2"
x git worktree remove "$T/wt-detached"
gap "gap a2 after remove"
lg -3

section "b: stash between verbs"
echo base > notes.txt; git add notes.txt; git commit -qm "chore: tracked notes" >/dev/null 2>&1
x git push -q origin main
gap "gap b1"
echo "wip" >> notes.txt; x git stash
echo "refs containing HEAD: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
gap "gap b2 (stash holds HEAD)"
x git stash pop
gap "gap b3 (after pop)"
lg -3
git checkout -q -- notes.txt

section "c: reset --soft HEAD~ after a run"
x git push -q origin main
gap "gap c1"; x aiwf set-priority $G low; x aiwf retitle $G "gap c1 retitled"
lg -2
x git reset --soft HEAD~
x git status --short
gap "gap c2 after soft reset"
x aiwf set-priority $G high
x git status --short
lg -2
section "c: user commits the staged leftovers with plain git commit"
x git commit -qm "chore: leftovers"
x aiwf check
x git reset -q --hard origin/main

section "d: reset --mixed HEAD~ after a run"
gap "gap d1"; x aiwf set-priority $G low
x git reset --mixed HEAD~
x git status --short
x aiwf set-priority $G high
x aiwf retitle $G "gap d1 retitled"
lg -2
x git reset -q --hard origin/main; git clean -qfd work

section "e: reset --hard HEAD~ after a run, then recovery from the reflog"
gap "gap e1"; x aiwf set-priority $G urgent; x aiwf retitle $G "gap e1 retitled"
x git reset --hard HEAD~
git reflog -6 --format='%h %gs'
x git reset --hard 'HEAD@{1}'
lg -1

section "f: pull --rebase with an unpushed amended run"
x git push -q origin main
gap "gap f1"; x aiwf set-priority $G low
peer_commit "peer-f"
x git pull -q --rebase origin main
lg -3
x aiwf set-priority $G high
lg -3
ops HEAD

section "g: pull (merge) with an unpushed amended run"
x git push -q origin main
gap "gap g1"
peer_commit "peer-g"
x git pull -q --no-rebase origin main
x aiwf set-priority $G low
lg -4

section "h: rebase of a branch whose tip is a planning commit"
x git push -q origin main
x git switch -q -c feat
echo f > f.txt; git add f.txt; git commit -qm "feat: f"
gap "gap h1 on feat"
x git switch -q main
peer_commit "peer-h"; x git pull -q --no-rebase origin main
x git switch -q feat
x git rebase -q main
lg -3
x aiwf set-priority $G low
lg -2
x git switch -q main

section "i: cherry-pick -x of a planning commit, then a verb"
x git push -q origin main
x git switch -q -c pick-src
gap "gap i1 on pick-src"
SRC=$(git rev-parse HEAD)
x git switch -q main
x git cherry-pick -x "$SRC"
echo "--- cherry-picked message:"; git log -1 --format=%B
x aiwf set-priority $G low
echo "--- after amend:"; git log -1 --format=%B
echo "cherry-pick marker still present after amend: $(git log -1 --format=%B | grep -c 'cherry picked from')"
x aiwf check
