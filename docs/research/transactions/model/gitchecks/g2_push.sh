#!/usr/bin/env bash
# Push and rebase facts for the transaction model.
set -euo pipefail
D=$(mktemp -d "${TMPDIR:-/tmp}/aiwf-gitcheck-push.XXXX"); cd "$D"
git init -q --bare remote.git
git clone -q remote.git a 2>/dev/null; cd a
git config user.name peter; git config user.email peter@example.com; git config commit.gpgsign false
echo 0 > f; git add f; git commit -q -m base; git push -q origin HEAD:main 2>/dev/null
git checkout -q -b tx
echo 1 >> f; git commit -qam one; git push -q -u origin tx 2>/dev/null
OLD=$(git rev-parse HEAD)
git commit -q --amend -m one-folded; NEW=$(git rev-parse HEAD)

echo "== G11a: lease-checked force push, remote already equals what we push, lease names the old SHA"
git push -q --force-with-lease=tx:$OLD origin tx 2>&1 | sed 's/^/   first push: /' || true
echo "   remote tx now = new? $([ "$(git ls-remote origin refs/heads/tx | cut -f1)" = "$NEW" ] && echo yes || echo no)"
out=$(git push --force-with-lease=tx:$OLD origin tx 2>&1) && rc=0 || rc=$?
echo "   retry with the same stale lease: exit $rc; $(echo "$out" | tr '\n' ' ' | cut -c1-160)"

echo "== G11b: lease-checked force push when another operator pushed meanwhile"
cd ..; git clone -q remote.git b 2>/dev/null; cd b
git config user.name bob; git config user.email bob@example.com; git config commit.gpgsign false
git checkout -q tx; echo bob >> f; git commit -qam bob; git push -q origin tx 2>/dev/null
cd ../a; git commit -q --amend -m one-folded-again
out=$(git push --force-with-lease=tx:$NEW origin tx 2>&1) && rc=0 || rc=$?
echo "   lease names the SHA a last pushed: exit $rc; $(echo "$out" | grep -o 'rejected.*' | head -1)"
git fetch -q origin
out=$(git push --force-with-lease origin tx 2>&1) && rc=0 || rc=$?
echo "   after a fetch, --force-with-lease with no explicit SHA: exit $rc; bob's commit still on remote? $(git log --format=%s "$(git ls-remote origin refs/heads/tx | cut -f1)" | grep -qx bob && echo yes || echo no)"

echo "== G12: git rebase keeps SHAs when the branch is already on the upstream tip"
cd ..; rm -rf c; git clone -q remote.git c 2>/dev/null; cd c
git config user.name peter; git config user.email peter@example.com; git config commit.gpgsign false
git checkout -q -b t2 origin/main; echo x > g; git add g; git commit -qm x; S=$(git rev-parse HEAD)
git rebase -q origin/main; echo "   up to date: SHA kept? $([ "$(git rev-parse HEAD)" = "$S" ] && echo yes || echo no)"
GIT_COMMITTER_DATE="2030-01-01T00:00:00Z" git rebase -q --force-rebase origin/main; echo "   --force-rebase: SHA kept? $([ "$(git rev-parse HEAD)" = "$S" ] && echo yes || echo no)"
