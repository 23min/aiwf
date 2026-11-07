#!/usr/bin/env bash
# Simulates a verb that commits to a side ref instead of the branch.
#   sideverb.sh <aiwf args...>
# Side chain: refs/aiwf/pending/<branch>, rooted at refs/aiwf/pending-base/<branch>
# (the branch tip when the chain started). Each side commit carries one verb's change
# and message. The branch itself never receives the verb's commit; the verb's files
# stay in the working tree, unstaged (index = branch tip).
# Mechanics: the branch is pointed for the verb's duration at a synthetic commit
# S = branch tip + pending net change (three-way), so the unmodified aiwf binary sees
# its own pending writes as committed; afterwards the branch goes back to its tip.
set -uo pipefail
B=$(git symbolic-ref --short HEAD) || exit 2
ref=refs/aiwf/pending/$B; bref=refs/aiwf/pending-base/$B
T=$(git rev-parse HEAD)
P=$(git rev-parse -q --verify "$ref")
paths=()
if [ -n "$P" ]; then
  B0=$(git rev-parse "$bref")
  out=$(git merge-tree --write-tree --merge-base="$B0" "$T" "$P") || { echo "sideverb: pending change no longer applies to the branch tip" >&2; echo "$out" >&2; exit 1; }
  S=$(echo synthetic | git commit-tree --no-gpg-sign "$(echo "$out" | head -1)" -p "$T")
  mapfile -t paths < <(git diff --name-only "$B0" "$P")
  git update-ref "refs/heads/$B" "$S" "$T"
  git reset -q "$S" -- "${paths[@]}" 2>/dev/null
else
  S=$T
fi
restore() { # branch back to T; index entries of every pending path back to T
  git update-ref "refs/heads/$B" "$T"
  [ ${#paths[@]} -gt 0 ] && git reset -q "$T" -- "${paths[@]}" 2>/dev/null
  return 0
}
aiwf "$@"; rc=$?
V=$(git rev-parse HEAD)
if [ "$V" = "$S" ]; then restore; exit $rc; fi
if [ -z "$P" ]; then git update-ref "$bref" "$T"; parent=$T; else parent=$P; fi
out=$(git merge-tree --write-tree --merge-base="$S" "$parent" "$V") || { echo "sideverb: cannot append to side chain" >&2; restore; exit 1; }
newP=$(git cat-file commit "$V" | sed '1,/^$/d' | git commit-tree --no-gpg-sign "$(echo "$out" | head -1)" -p "$parent")
git update-ref "$ref" "$newP"
mapfile -t paths < <(git diff --name-only "$(git rev-parse "$bref")" "$newP")
restore
echo "sideverb: recorded on $ref ($(git rev-parse --short "$newP")); branch stays at $(git rev-parse --short "$T")"
exit $rc
