#!/usr/bin/env bash
# Does git rebase drop a branch commit whose change already reached upstream through a squash?
set -euo pipefail
D=$(mktemp -d "${TMPDIR:-/tmp}/aiwf-gitcheck-empty.XXXX"); cd "$D"
git init -q -b main .; git config user.name peter; git config user.email peter@example.com; git config commit.gpgsign false
echo 0 > f; echo 0 > g; git add f g; git commit -qm base
git checkout -q -b tx; echo 1 > f; git commit -qam "plan: Q"; echo 1 > g; git commit -qam "feat: code"
git checkout -q main; git merge -q --squash tx >/dev/null; git commit -qm "squash of tx"
git checkout -q tx; echo 2 > f; git commit -qam "plan: verb after the merge"
git rebase -q main 2>&1 | sed 's/^/   rebase: /' || true
echo "G13 commits left on tx after rebasing onto the squash: $(git log --format=%s main..tx | tr '\n' '|')"
echo "== G14: does git rebase keep a commit that started empty (an aiwf authorize commit)?"
git checkout -q -b scoped "$(git rev-list --max-parents=0 HEAD | tail -1)"
git commit -q --allow-empty -m "aiwf authorize M-0001" -m "aiwf-verb: authorize
aiwf-entity: M-0001
aiwf-scope: opened"
E=$(git rev-parse HEAD)
echo 3 > h; git add h; git commit -q -m "feat: agent work"
git rebase -q main 2>&1 | sed 's/^/   rebase: /' || true
N=$(git log --format=%H --grep='aiwf authorize' main..scoped | head -1)
echo "G14 empty authorize commit after rebase: present=$([ -n "$N" ] && echo yes || echo no) same SHA=$([ "$N" = "$E" ] && echo yes || echo no) trailers parsed=$(git log -1 --format='%(trailers:key=aiwf-verb,valueonly=true)' "$N" | grep -c .)"
