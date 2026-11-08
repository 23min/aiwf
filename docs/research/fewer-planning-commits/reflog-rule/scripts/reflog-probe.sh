#!/usr/bin/env bash
# Which git commands write the top line of HEAD's reflog, and what it says; and the
# rebase hole: a pushed verb commit, rebased, is contained by no ref but its own branch.
# A "verb" here commits as aiwf does (commit-tree, then update-ref of HEAD).
# Usage: reflog-probe.sh <scratch dir>   (the directory is emptied)
set -eu
S=${1:?scratch dir}; rm -rf "$S"; mkdir -p "$S/home"; cd "$S"
export HOME=$S/home GIT_CONFIG_NOSYSTEM=1 GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@example.com \
       GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@example.com
git config --global init.defaultBranch main; git config --global pull.rebase false
git --version
git init -q --bare remote.git; git init -q w; cd w; git remote add origin ../remote.git
echo "core.logAllRefUpdates in a new repo: $(git config --get core.logAllRefUpdates)"
top(){ printf '%-44s HEAD@{0}: %s\n' "$1" "$(git reflog -1 HEAD --format='%gs')"; }
verb(){ echo "$1" >"$1"; git add "$1"; t=$(git write-tree); c=$(git commit-tree "$t" -p HEAD -m "aiwf $1")
        git update-ref ${2:+-m "$2"} HEAD "$c" HEAD; }
echo a >a; git add a; git commit -qm init; top "git commit"
git push -q -u origin main
verb v0;                     top "verb, update-ref without -m (aiwf today)"
verb v1 "aiwf: v1";          top "verb, update-ref -m 'aiwf: v1'"
git branch side;             top "git branch side (no switch)"
git push -q origin main;     top "git push"
git fetch -q;                top "git fetch"
git switch -q side;          top "git switch side"
git switch -q main;          top "git switch main"
echo s >s; git add s; git stash -q; top "git stash"
git stash pop -q;            top "git stash pop (HEAD unchanged)"
git commit -qm c1;           top "git commit"
git commit -q --amend -m c2; top "git commit --amend"
git reset -q --soft HEAD~1;  top "git reset --soft HEAD~1"
git merge -q --no-edit side; top "git merge side (up to date; HEAD unchanged)"
echo
echo "== the rebase hole =="
git switch -q -c milestone; verb m1 "aiwf: m1"; git push -q -u origin milestone 2>/dev/null
top "verb m1, pushed on branch milestone"
git clone -q ../remote.git ../other
(cd ../other && git switch -q main && echo o >o && git add o && git commit -qm other && git push -q origin main)
git fetch -q
git rebase -q origin/main;   top "git rebase origin/main"
echo "pushed m1: $(git rev-parse --short origin/milestone)   HEAD: $(git rev-parse --short HEAD)"
echo "refs containing HEAD: $(git for-each-ref --contains HEAD --format='%(refname)' | tr '\n' ' ')"
echo "git cherry origin/milestone HEAD ('-' = the same change is upstream): $(git cherry origin/milestone HEAD | cut -c1-9 | tr '\n' ' ')"
verb m2 "aiwf: m2";          top "next verb (the reflog rule commits anew)"
git switch -q main; git pull -q; top "git pull (fast-forward)"
(cd ../other && echo p >p && git add p && git commit -qm other2 && git push -q origin main)
verb x1 "aiwf: x1"; git pull -q --no-edit; top "git pull (merge)"
git switch -q -c pick main; git cherry-pick milestone >/dev/null; top "git cherry-pick"
git switch -q milestone
git pull -q --rebase origin main; top "git pull --rebase"
