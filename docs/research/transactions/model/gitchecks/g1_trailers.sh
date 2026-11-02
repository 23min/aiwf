#!/usr/bin/env bash
# G-checks for the transaction model. Each prints COMMAND / OBSERVED lines.
set -euo pipefail
D=$(mktemp -d "${TMPDIR:-/tmp}/aiwf-gitcheck-repo.XXXX"); cd "$D"
export GIT_AUTHOR_DATE='2026-10-26T10:00:00Z' GIT_COMMITTER_DATE='2026-10-26T10:00:00Z'
git init -q -b main .; git config user.name peter; git config user.email peter@example.com
git config commit.gpgsign false
echo base > f; git add f; git commit -q -m base
ntr() { git log -1 --format='%(trailers:key=aiwf-verb,valueonly=true)' "$1" | grep -c . || true; }

echo "== G7: trailers parse only from the last paragraph"
echo a >> f; git add f
git commit -q -F - <<'M'
subject

aiwf-verb: add
aiwf-entity: G-0001

aiwf-verb: promote
aiwf-entity: M-0001
M
echo "two trailer paragraphs -> aiwf-verb values parsed: $(ntr HEAD) ($(git log -1 --format='%(trailers:key=aiwf-verb,valueonly=true,separator=%x2C)'))"
echo b >> f; git add f
git commit -q -F - <<'M'
subject

aiwf-verb: add
aiwf-entity: G-0001

Co-authored-by: bob <bob@example.com>
M
echo "trailers then a co-author paragraph -> aiwf-verb values parsed: $(ntr HEAD)"
echo c >> f; git add f
git commit -q -F - <<'M'
subject

aiwf-verb: add
aiwf-entity: G-0001
aiwf-actor: human/peter
aiwf-verb: promote
aiwf-entity: M-0001
aiwf-actor: human/bob
M
echo "one paragraph, two groups of repeated keys -> aiwf-verb values parsed: $(ntr HEAD)"

echo "== G8: git merge --squash default message"
git checkout -q -b side
echo d >> f; git add f; git commit -q -m "s1" -m "aiwf-verb: add
aiwf-entity: G-0002"
git checkout -q main; git merge -q --squash side >/dev/null 2>&1
git commit -q --no-edit
echo "squash default message, last 3 lines:"; git log -1 --format=%B | tail -3 | sed 's/^/   |/'
echo "aiwf-verb values parsed from it: $(ntr HEAD)"

echo "== G9: cherry-pick keeps the trailer block parseable?"
git checkout -q -b cp main~1
echo e > g; git add g; git commit -q -m "c0"
git cherry-pick -x side >/dev/null
echo "cherry-pick -x: message tail:"; git log -1 --format=%B | tail -3 | sed 's/^/   |/'
echo "aiwf-verb values parsed: $(ntr HEAD)"
git reset -q --hard HEAD~1
git cherry-pick side >/dev/null
echo "cherry-pick without -x: aiwf-verb values parsed: $(ntr HEAD); same SHA as source? $([ "$(git rev-parse HEAD)" = "$(git rev-parse side)" ] && echo yes || echo no)"

echo "== G10: revert message"
git revert --no-edit HEAD >/dev/null
echo "revert: aiwf-verb values parsed: $(ntr HEAD); message: $(git log -1 --format=%B | tr '\n' ' ')"
