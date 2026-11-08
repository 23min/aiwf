#!/usr/bin/env bash
# S09: edges of the eligibility rule.
#  a: --by-commit naming HEAD (corrected vs naive rule)
#  b: --by-commit naming an earlier op's commit_sha that a later amend made unreachable
#  c: a hand-made `git commit --trailer aiwf-verb: ...` carrying code, then a verb
#  d: a ritual-stamped commit (wrap-milestone), then a verb
. "$(dirname "$0")/lib.sh"
new_env s09; init_repo repo
GB=$'## What\'s missing\n\nx.\n\n## Why it matters\n\ny.'
gapt() { x aiwf add gap --title "$1" --body "$GB"; G=$(ls work/gaps | sort | tail -1 | cut -c1-6); }

section "a: --by-commit naming HEAD"
gapt "gap a"; GA=$G
gapt "gap a2"; GA2=$G
x aiwf promote "$GA" addressed --by-commit "$(git rev-parse HEAD)"
lg -2
gapt "gap a3"; GA3=$G
AIWF_EXPERIMENT_AMEND_RULE=naive x aiwf promote "$GA3" addressed --by-commit "$(git rev-parse HEAD)"
lg -1
SHA=$(grep -h 'addressed_by_commit' -A1 work/gaps/"$GA3"-*.md | grep -o '[0-9a-f]\{7,40\}' | head -1)
echo "addressed_by_commit of $GA3: $SHA; reachable from HEAD: $(git merge-base --is-ancestor "$SHA" HEAD 2>/dev/null && echo yes || echo NO)"
x aiwf check

section "b: an earlier op's commit_sha, made unreachable by a later amend, used as --by-commit"
x git push -q origin main
OUT=$(aiwf add gap --title "gap b1" --body "$GB" --format=json 2>/dev/null); echo "$OUT" | python3 -I -c 'import json,sys; d=json.load(sys.stdin); print("add printed commit_sha:", d["metadata"].get("commit_sha"))'
SHA1=$(echo "$OUT" | python3 -I -c 'import json,sys; print(json.load(sys.stdin)["metadata"]["commit_sha"])')
GB1=$(ls work/gaps | sort | tail -1 | cut -c1-6)
gapt "gap b2"
echo "is $SHA1 an ancestor of HEAD now? $(git merge-base --is-ancestor "$SHA1" HEAD && echo yes || echo NO)"
x aiwf promote "$GB1" addressed --by-commit "$SHA1"
lg -2
x aiwf check
x git push origin main
x git gc -q --prune=now
echo "object $SHA1 after gc: $(git cat-file -t "$SHA1" 2>&1)"
x aiwf show "$GB1"

section "c: hand-made commit with kernel-verb trailers and a code change, then a verb"
x git push -q origin main
echo "code" > code.txt; git add code.txt
x git commit -q -m "fix: code change" --trailer "aiwf-verb: promote" --trailer "aiwf-actor: human/peter"
lg -1
gapt "gap c"
lg -2
echo "--- files in HEAD's diff:"; git show --stat --format=%s HEAD | tail -3

section "d: ritual-stamped commit, then a verb"
x git push -q origin main
echo "wrap notes" > wrap.md; git add wrap.md
x git commit -q -m "chore: wrap" --trailer "aiwf-verb: wrap-milestone" --trailer "aiwf-actor: human/peter"
gapt "gap d"
lg -2
