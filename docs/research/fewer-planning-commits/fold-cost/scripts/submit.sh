#!/usr/bin/env bash
# Lands the side chain's net change on the branch tip as one commit (three-way, not a
# tree copy), carrying each side commit's trailers in order; then deletes the side refs.
#   submit.sh
set -uo pipefail
B=$(git symbolic-ref --short HEAD) || exit 2
ref=refs/aiwf/pending/$B; bref=refs/aiwf/pending-base/$B
P=$(git rev-parse -q --verify "$ref") || { echo "submit: nothing pending"; exit 0; }
B0=$(git rev-parse "$bref"); T=$(git rev-parse HEAD)
out=$(git merge-tree --write-tree --merge-base="$B0" "$T" "$P") || { echo "submit: conflict" >&2; echo "$out" >&2; exit 1; }
tree=$(echo "$out" | head -1)
subjects=""; trailers=""; n=0
for c in $(git rev-list --reverse "$B0..$P"); do
  n=$((n+1)); subjects+="- $(git log -1 --format=%s "$c")"$'\n'
  trailers+="$(git log -1 --format='%(trailers:only=true,unfold=true)' "$c" | sed '/^$/d')"$'\n'
done
msg="chore(plan): submit $n planning operations"$'\n\n'"$subjects"$'\n'"$trailers"
new=$(printf '%s' "$msg" | git commit-tree "$tree" -p "$T")
git update-ref "refs/heads/$B" "$new" "$T"
git update-ref -d "$ref"; git update-ref -d "$bref"
git update-index -q --refresh >/dev/null; git reset -q
echo "submit: $n operations -> $(git log -1 --format='%h' "$new"); changed vs tip: $(git diff --name-only "$T" "$new" | tr '\n' ' ')"
